package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm/prompts"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// SendMessage persists the user message, streams the assistant reply to sink,
// and persists the result. Errors before streaming (validation, rate limit)
// are returned so the handler can answer with JSON; once streaming starts,
// failures are emitted via sink and the assistant message is finalized
// (complete / failed / cancelled).
func (s *Service) SendMessage(ctx context.Context, userID uuid.UUID, in SendInput, sink Sink) error {
	userText, err := validateInput(in)
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

	conv, err := s.store.GetOrCreateConversation(ctx, userID)
	if err != nil {
		return fmt.Errorf("chat: conversation: %w", err)
	}

	if err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		m, err := q.CreateMessage(ctx, db.CreateMessageParams{
			ConversationID: conv.ID, UserID: userID, Role: "user", Status: "complete",
		})
		if err != nil {
			return err
		}
		_, err = q.CreateMessageBlock(ctx, db.CreateMessageBlockParams{
			MessageID: m.ID, OrderIndex: 0, Type: "text", ContentText: textVal(userText),
		})
		return err
	}); err != nil {
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

	if err := sink.MessageStarted(assistantID.String()); err != nil {
		s.finalizeIncomplete(assistantID, "cancelled", "")
		return err
	}

	req := llm.SendRequest{
		Model:    s.model,
		System:   prompts.SystemV1(),
		Messages: history,
		Metadata: llm.Metadata{UIDHash: uidHash(userID, s.pepper), RequestID: in.RequestID},
	}
	ch, err := s.llm.Send(ctx, req)
	if err != nil {
		sink.Failed("upstream_error", "the assistant is unavailable")
		s.finalizeIncomplete(assistantID, "failed", "")
		return fmt.Errorf("chat: llm send: %w", err)
	}

	res, disconnectErr := relay(ch, sink)
	switch {
	case disconnectErr != nil: // client went away mid-stream
		s.finalizeIncomplete(assistantID, "cancelled", res.text)
		return disconnectErr
	case ctx.Err() != nil: // request context cancelled
		s.finalizeIncomplete(assistantID, "cancelled", res.text)
		return ctx.Err()
	case res.failed: // upstream error already sent to the client
		s.finalizeIncomplete(assistantID, "failed", res.text)
		return nil
	default:
		s.finalizeComplete(assistantID, userID, res.text, res.tokensIn, res.tokensOut)
		return sink.Done(assistantID.String(), res.tokensIn, res.tokensOut)
	}
}

type relayResult struct {
	text                string
	tokensIn, tokensOut int64
	failed              bool
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
			_ = sink.FertilizerCard(ev.Data)
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
// context (so the writes land even if the request context is done).
func (s *Service) finalizeComplete(assistantID, userID uuid.UUID, text string, in, out int64) {
	fctx, cancel := context.WithTimeout(context.Background(), finalizeTimeout)
	defer cancel()

	err := s.store.ExecTx(fctx, func(q *db.Queries) error {
		if err := q.CompleteMessage(fctx, db.CompleteMessageParams{
			ID: assistantID, TokensIn: int4(in), TokensOut: int4(out),
		}); err != nil {
			return err
		}
		if text != "" {
			if _, err := q.CreateMessageBlock(fctx, db.CreateMessageBlockParams{
				MessageID: assistantID, OrderIndex: 0, Type: "text", ContentText: textVal(text),
			}); err != nil {
				return err
			}
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
