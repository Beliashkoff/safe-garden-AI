package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/audio"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm/prompts"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/observability"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// SendMessage persists the user message, streams the assistant reply to sink,
// and persists the result. Errors before streaming (validation, rate limit)
// are returned so the handler can answer with JSON; once streaming starts,
// failures are emitted via sink and the assistant message is finalized
// (complete / failed / cancelled).
func (s *Service) SendMessage(ctx context.Context, userID uuid.UUID, in SendInput, sink Sink) error {
	blocks, err := validateInput(userID, in)
	if err != nil {
		return err
	}

	allowed, err := s.limiter.AllowMessage(ctx, userID)
	if err != nil {
		return fmt.Errorf("chat: rate check: %w", err)
	}
	if !allowed {
		return ErrRateLimited
	}

	if err := s.verifyMediaRefs(ctx, userID, blocks); err != nil {
		return err
	}

	// Transcribe voice messages before anything is persisted or streamed, so a
	// failure surfaces as a pre-stream JSON error and nothing is saved.
	if err := s.transcribeAudio(ctx, blocks); err != nil {
		return err
	}

	conv, err := s.store.GetOrCreateConversation(ctx, userID)
	if err != nil {
		return fmt.Errorf("chat: conversation: %w", err)
	}

	userMsgID, err := s.saveUserMessage(ctx, conv.ID, userID, blocks)
	if err != nil {
		return fmt.Errorf("chat: save user message: %w", err)
	}

	history, err := s.loadHistory(ctx, conv.ID)
	if err != nil {
		return fmt.Errorf("chat: load history: %w", err)
	}

	assistant, err := s.store.CreateMessage(ctx, db.CreateMessageParams{
		ConversationID: conv.ID, UserID: userID, Role: "assistant", Status: "pending",
	})
	if err != nil {
		return fmt.Errorf("chat: create assistant message: %w", err)
	}
	assistantID := assistant.ID

	// First sink writes: surface each voice transcription to the client before
	// the assistant reply begins. A write error means the client disconnected.
	for _, b := range blocks {
		if b.kind != "audio" {
			continue
		}
		if err := sink.Transcription(userMsgID.String(), b.storageKey, b.transcriptText, b.durationMs); err != nil {
			s.finalizeIncomplete(assistantID, "cancelled", "")
			return err
		}
	}

	if err := sink.MessageStarted(assistantID.String()); err != nil {
		s.finalizeIncomplete(assistantID, "cancelled", "")
		return err
	}

	req := llm.SendRequest{
		Model:    s.model,
		System:   prompts.SystemV1(),
		Messages: history,
		Tools:    llm.FertilizerTools(),
		Metadata: llm.Metadata{UIDHash: uidHash(userID, s.pepper), RequestID: in.RequestID},
	}
	start := time.Now()
	ch, err := s.llm.Send(ctx, req)
	if err != nil {
		observability.ObserveClaudeTurn(0, 0, 0, time.Since(start), "upstream_error")
		observability.IncMessage("failed")
		sink.Failed("upstream_error", "the assistant is unavailable")
		s.finalizeIncomplete(assistantID, "failed", "")
		return fmt.Errorf("chat: llm send: %w", err)
	}

	res, disconnectErr := relay(ch, sink)
	dur := time.Since(start)
	cost := llm.EstimateCostUSD(s.model, res.tokensIn, res.tokensOut)
	switch {
	case disconnectErr != nil: // client went away mid-stream — not a Claude error
		observability.ObserveClaudeTurn(res.tokensIn, res.tokensOut, cost, dur, "")
		observability.IncMessage("cancelled")
		s.finalizeIncomplete(assistantID, "cancelled", res.text)
		return disconnectErr
	case ctx.Err() != nil: // request context cancelled — not a Claude error
		observability.ObserveClaudeTurn(res.tokensIn, res.tokensOut, cost, dur, "")
		observability.IncMessage("cancelled")
		s.finalizeIncomplete(assistantID, "cancelled", res.text)
		return ctx.Err()
	case res.failed: // upstream error already sent to the client
		observability.ObserveClaudeTurn(res.tokensIn, res.tokensOut, cost, dur, "upstream_error")
		observability.IncMessage("failed")
		s.finalizeIncomplete(assistantID, "failed", res.text)
		return nil
	default:
		observability.ObserveClaudeTurn(res.tokensIn, res.tokensOut, cost, dur, "")
		observability.IncMessage("complete")
		s.finalizeComplete(assistantID, userID, res.text, res.fertilizerCards, res.tokensIn, res.tokensOut)
		return sink.Done(assistantID.String(), res.tokensIn, res.tokensOut)
	}
}

type relayResult struct {
	text                string
	tokensIn, tokensOut int64
	failed              bool
	// fertilizerCards holds the raw {"products":[...]} payloads emitted during
	// the turn, in order, so they can be persisted as fertilizer_card blocks.
	fertilizerCards []json.RawMessage
}

// relay forwards worker stream events to the sink and accumulates the assistant
// text + usage. It returns a non-nil error only when a sink write fails (client
// disconnect); upstream model errors are reported via res.failed.
func relay(ch <-chan llm.StreamEvent, sink Sink) (relayResult, error) {
	var acc strings.Builder
	var res relayResult
	for ev := range ch {
		switch ev.Type {
		case llm.EventDelta:
			var d struct {
				Text string `json:"text"`
			}
			_ = json.Unmarshal(ev.Data, &d)
			if d.Text == "" {
				continue
			}
			acc.WriteString(d.Text)
			if err := sink.Delta(d.Text); err != nil {
				res.text = acc.String()
				return res, err
			}
		case llm.EventToolUse:
			var t struct {
				Tool string          `json:"tool"`
				Args json.RawMessage `json:"args"`
			}
			_ = json.Unmarshal(ev.Data, &t)
			_ = sink.ToolUse(t.Tool, t.Args)
		case llm.EventFertilizerCard:
			// Copy: ev.Data is backed by the stream buffer, reused on the next read.
			data := make(json.RawMessage, len(ev.Data))
			copy(data, ev.Data)
			res.fertilizerCards = append(res.fertilizerCards, data)
			_ = sink.FertilizerCard(data)
		case llm.EventUsage:
			var u struct {
				In  int64 `json:"tokens_in"`
				Out int64 `json:"tokens_out"`
			}
			_ = json.Unmarshal(ev.Data, &u)
			res.tokensIn, res.tokensOut = u.In, u.Out
		case llm.EventError:
			var e struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			_ = json.Unmarshal(ev.Data, &e)
			sink.Failed(orDefault(e.Code, "upstream_error"), orDefault(e.Message, "the assistant failed"))
			res.failed = true
		}
	}
	res.text = acc.String()
	return res, nil
}

// finalizeComplete persists the finished assistant message + usage on a detached
// context (so the writes land even if the request context is done). Any
// fertilizer cards emitted during the turn are stored as fertilizer_card blocks
// after the text block, so reloading history restores them (ARCH §6.1).
func (s *Service) finalizeComplete(assistantID, userID uuid.UUID, text string, cards []json.RawMessage, in, out int64) {
	fctx, cancel := context.WithTimeout(context.Background(), finalizeTimeout)
	defer cancel()

	err := s.store.ExecTx(fctx, func(q *db.Queries) error {
		if err := q.CompleteMessage(fctx, db.CompleteMessageParams{
			ID: assistantID, TokensIn: int4(in), TokensOut: int4(out),
		}); err != nil {
			return err
		}
		var order int32
		if text != "" {
			if _, err := q.CreateMessageBlock(fctx, db.CreateMessageBlockParams{
				MessageID: assistantID, OrderIndex: order, Type: "text", ContentText: textVal(text),
			}); err != nil {
				return err
			}
			order++
		}
		for _, card := range cards {
			if _, err := q.CreateMessageBlock(fctx, db.CreateMessageBlockParams{
				MessageID: assistantID, OrderIndex: order, Type: "fertilizer_card", Metadata: card,
			}); err != nil {
				return err
			}
			order++
		}
		return nil
	})
	if err != nil {
		s.logger.Error("chat: finalize complete failed", "err", err.Error())
	}
	if err := s.store.InsertUsage(fctx, db.InsertUsageParams{
		UserID: userID, Endpoint: "/v1/messages", TokensIn: int4(in), TokensOut: int4(out),
	}); err != nil {
		s.logger.Error("chat: usage insert failed", "err", err.Error())
	}
}

// finalizeIncomplete marks the assistant message cancelled/failed and saves any
// partial text, on a detached context.
func (s *Service) finalizeIncomplete(assistantID uuid.UUID, status, partial string) {
	fctx, cancel := context.WithTimeout(context.Background(), finalizeTimeout)
	defer cancel()

	err := s.store.ExecTx(fctx, func(q *db.Queries) error {
		if err := q.UpdateMessageStatus(fctx, db.UpdateMessageStatusParams{ID: assistantID, Status: status}); err != nil {
			return err
		}
		if partial != "" {
			if _, err := q.CreateMessageBlock(fctx, db.CreateMessageBlockParams{
				MessageID: assistantID, OrderIndex: 0, Type: "text", ContentText: textVal(partial),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.logger.Error("chat: finalize incomplete failed", "status", status, "err", err.Error())
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// verifyMediaRefs confirms each image_ref/audio_ref points at an upload owned by
// the caller with an allowed content type (pre-stream ownership check, ARCH §8.2).
func (s *Service) verifyMediaRefs(ctx context.Context, userID uuid.UUID, blocks []validatedBlock) error {
	for _, b := range blocks {
		var whitelist map[string]struct{}
		switch b.kind {
		case "image":
			whitelist = imageContentTypes
		case "audio":
			whitelist = audioContentTypes
		default:
			continue
		}
		up, err := s.store.GetUploadByStorageKey(ctx, b.storageKey)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUploadNotFound
			}
			return fmt.Errorf("chat: lookup upload: %w", err)
		}
		if up.UserID != userID {
			return ErrUploadNotFound
		}
		if _, ok := whitelist[up.ContentType]; !ok {
			return ErrUnsupportedBlock
		}
	}
	return nil
}

// transcribeAudio downloads, converts (ffmpeg → OggOpus), and transcribes each
// audio block, enriching it in place with the transcript and duration. Called
// before any persistence/streaming so failures return as pre-stream errors.
func (s *Service) transcribeAudio(ctx context.Context, blocks []validatedBlock) error {
	for i := range blocks {
		if blocks[i].kind != "audio" {
			continue
		}
		data, contentType, err := s.images.GetLimited(ctx, blocks[i].storageKey, maxAudioReadBytes)
		if err != nil {
			s.logger.ErrorContext(ctx, "chat: audio fetch failed", "err", err.Error())
			return ErrTranscriptionFailed
		}
		ogg, durationMs, err := s.audioConv.ToOggOpus(ctx, data, contentType)
		if err != nil {
			if errors.Is(err, audio.ErrTooLong) {
				return ErrAudioTooLong
			}
			s.logger.ErrorContext(ctx, "chat: audio convert failed", "err", err.Error())
			return ErrTranscriptionFailed
		}

		tctx, cancel := context.WithTimeout(ctx, transcribeTimeout)
		res, err := s.stt.Transcribe(tctx, ogg, s.lang)
		cancel()
		if err != nil {
			if errors.Is(err, audio.ErrEmptyResult) {
				return ErrTranscriptionEmpty
			}
			s.logger.ErrorContext(ctx, "chat: transcription failed", "err", err.Error())
			return ErrTranscriptionFailed
		}

		blocks[i].transcriptText = res.Text
		blocks[i].durationMs = durationMs // converter duration is authoritative
	}
	return nil
}

// saveUserMessage persists the user message and its blocks (text + image +
// audio/transcription), marking referenced uploads used, in one transaction. It
// returns the new user message id (for the transcription SSE event).
func (s *Service) saveUserMessage(ctx context.Context, convID, userID uuid.UUID, blocks []validatedBlock) (uuid.UUID, error) {
	var msgID uuid.UUID
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		m, err := q.CreateMessage(ctx, db.CreateMessageParams{
			ConversationID: convID, UserID: userID, Role: "user", Status: "complete",
		})
		if err != nil {
			return err
		}
		msgID = m.ID
		var order int32
		for _, b := range blocks {
			n, err := saveBlock(ctx, q, m.ID, order, b)
			if err != nil {
				return err
			}
			order += n
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return msgID, nil
}

// saveBlock persists one input block and returns how many message_blocks rows it
// wrote, so the caller advances order_index. An empty text block writes nothing;
// an audio block writes two rows (the original audio + its transcription).
func saveBlock(ctx context.Context, q *db.Queries, msgID uuid.UUID, order int32, b validatedBlock) (int32, error) {
	switch b.kind {
	case "text":
		if strings.TrimSpace(b.text) == "" {
			return 0, nil
		}
		_, err := q.CreateMessageBlock(ctx, db.CreateMessageBlockParams{
			MessageID: msgID, OrderIndex: order, Type: "text", ContentText: textVal(b.text),
		})
		return 1, err
	case "image":
		if _, err := q.CreateMessageBlock(ctx, db.CreateMessageBlockParams{
			MessageID: msgID, OrderIndex: order, Type: "image", StorageKey: textVal(b.storageKey),
		}); err != nil {
			return 0, err
		}
		return 1, q.MarkUploadUsed(ctx, b.storageKey)
	case "audio":
		if _, err := q.CreateMessageBlock(ctx, db.CreateMessageBlockParams{
			MessageID: msgID, OrderIndex: order, Type: "audio", StorageKey: textVal(b.storageKey),
		}); err != nil {
			return 0, err
		}
		if _, err := q.CreateMessageBlock(ctx, db.CreateMessageBlockParams{
			MessageID: msgID, OrderIndex: order + 1, Type: "transcription",
			ContentText: textVal(b.transcriptText), Metadata: durationMeta(b.durationMs),
		}); err != nil {
			return 0, err
		}
		return 2, q.MarkUploadUsed(ctx, b.storageKey)
	}
	return 0, nil
}
