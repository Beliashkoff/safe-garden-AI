package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/ctxkey"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
	chatuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/chat"
)

// --- request / response DTOs (ARCH §4.3) ---

type postMessageRequest struct {
	Content []contentBlockReq `json:"content"`
}

type contentBlockReq struct {
	Type       string `json:"type"`
	Text       string `json:"text"`
	StorageKey string `json:"storage_key"`
}

type blockDTO struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	StorageKey string `json:"storage_key,omitempty"`
}

type messageDTO struct {
	ID        string     `json:"id"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	Content   []blockDTO `json:"content"`
}

type conversationDTO struct {
	ID         string       `json:"id"`
	Messages   []messageDTO `json:"messages"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

type messagesPageDTO struct {
	Messages   []messageDTO `json:"messages"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

// PostMessage handles POST /v1/messages — streams the assistant reply as SSE.
func (h *Handler) PostMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxkey.UserID(r.Context())
	if !ok {
		httperr.Write(w, r, httperr.Unauthorized("authentication required"))
		return
	}
	var req postMessageRequest
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}

	in := chatuc.SendInput{
		Blocks:    toInputBlocks(req.Content),
		RequestID: chimw.GetReqID(r.Context()),
	}
	sink := &sseSink{w: w}
	if err := h.chat.SendMessage(r.Context(), userID, in, sink); err != nil {
		if sink.started {
			// Headers already flushed; any client-facing error was emitted as an
			// SSE error event (or the client simply disconnected). Nothing to add.
			return
		}
		respondError(w, r, err)
	}
}

// GetConversation handles GET /v1/conversation — latest page of history.
func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxkey.UserID(r.Context())
	if !ok {
		httperr.Write(w, r, httperr.Unauthorized("authentication required"))
		return
	}
	view, err := h.chat.GetConversation(r.Context(), userID, parseLimit(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, conversationDTO{
		ID:         view.ConversationID,
		Messages:   toMessageDTOs(view.Messages),
		NextCursor: view.NextCursor,
	})
}

// ListMessages handles GET /v1/conversation/messages?cursor=&limit= — older pages.
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxkey.UserID(r.Context())
	if !ok {
		httperr.Write(w, r, httperr.Unauthorized("authentication required"))
		return
	}
	cursor := r.URL.Query().Get("cursor")
	if cursor == "" {
		httperr.Write(w, r, httperr.ValidationFailed("cursor is required; use GET /conversation for the first page"))
		return
	}
	page, err := h.chat.ListMessages(r.Context(), userID, cursor, parseLimit(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, messagesPageDTO{
		Messages:   toMessageDTOs(page.Messages),
		NextCursor: page.NextCursor,
	})
}

// DeleteMessage handles DELETE /v1/messages/{id} — owner-scoped.
func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxkey.UserID(r.Context())
	if !ok {
		httperr.Write(w, r, httperr.Unauthorized("authentication required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httperr.Write(w, r, httperr.ValidationFailed("invalid message id"))
		return
	}
	if err := h.chat.DeleteMessage(r.Context(), userID, id); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toInputBlocks(in []contentBlockReq) []chatuc.InputBlock {
	out := make([]chatuc.InputBlock, len(in))
	for i, b := range in {
		out[i] = chatuc.InputBlock{Type: b.Type, Text: b.Text, StorageKey: b.StorageKey}
	}
	return out
}

func toMessageDTOs(views []chatuc.MessageView) []messageDTO {
	out := make([]messageDTO, len(views))
	for i, v := range views {
		blocks := make([]blockDTO, len(v.Content))
		for j, b := range v.Content {
			blocks[j] = blockDTO{Type: b.Type, Text: b.Text, StorageKey: b.StorageKey}
		}
		out[i] = messageDTO{ID: v.ID, Role: v.Role, Status: v.Status, CreatedAt: v.CreatedAt, Content: blocks}
	}
	return out
}

func parseLimit(r *http.Request) int {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		return 0 // usecase clamps to default
	}
	return n
}

// sseSink adapts chat.Sink onto the HTTP response as SSE (ARCH §4.3 events).
// Headers are written lazily on the first event so pre-stream errors
// (validation, rate limit) can still be answered with a JSON body.
type sseSink struct {
	w       http.ResponseWriter
	started bool
}

func (s *sseSink) ensure() {
	if !s.started {
		setSSEHeaders(s.w)
		s.w.WriteHeader(http.StatusOK)
		s.started = true
	}
}

func (s *sseSink) MessageStarted(messageID string) error {
	s.ensure()
	return writeSSE(s.w, "message_started", map[string]string{"message_id": messageID})
}

func (s *sseSink) Delta(text string) error {
	return writeSSE(s.w, "delta", map[string]string{"text": text})
}

func (s *sseSink) ToolUse(name string, args json.RawMessage) error {
	return writeSSE(s.w, "tool_use", map[string]any{"tool": name, "args": args})
}

func (s *sseSink) FertilizerCard(data json.RawMessage) error {
	return writeSSE(s.w, "fertilizer_card", data)
}

func (s *sseSink) Done(messageID string, tokensIn, tokensOut int64) error {
	return writeSSE(s.w, "done", map[string]any{
		"message_id":  messageID,
		"tokens_used": map[string]int64{"in": tokensIn, "out": tokensOut},
	})
}

func (s *sseSink) Failed(code, msg string) {
	s.ensure()
	_ = writeSSE(s.w, "error", map[string]string{"code": code, "message": msg})
}
