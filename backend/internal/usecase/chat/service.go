package chat

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

const (
	maxTextBytes     = 32 * 1024 // ARCH §8.2 text body cap
	historyLimit     = 20        // ARCH §7.4 context window
	finalizeTimeout  = 15 * time.Second
	defaultPageLimit = 50
	maxPageLimit     = 100
)

// messageLimiter is the per-user rate gate (consumer-side interface; satisfied
// by ratelimit.RedisLimiter / NoopMessage).
type messageLimiter interface {
	AllowMessage(ctx context.Context, userID uuid.UUID) (bool, error)
}

// Service orchestrates chat persistence + the LLM client.
type Service struct {
	store   *storage.Store
	llm     llm.Client
	limiter messageLimiter
	pepper  string
	model   string
	logger  *slog.Logger
	now     func() time.Time
}

func NewService(
	store *storage.Store,
	client llm.Client,
	limiter messageLimiter,
	pepper, model string,
	logger *slog.Logger,
) *Service {
	return &Service{
		store:   store,
		llm:     client,
		limiter: limiter,
		pepper:  pepper,
		model:   model,
		logger:  logger,
		now:     time.Now,
	}
}

// validateInput flattens the request blocks into a single user text, enforcing
// the text-only + size rules of stage 2.3.
func validateInput(in SendInput) (string, error) {
	if len(in.Blocks) == 0 {
		return "", ErrEmptyContent
	}
	var b strings.Builder
	for _, blk := range in.Blocks {
		if blk.Type != "text" {
			return "", ErrUnsupportedBlock
		}
		b.WriteString(blk.Text)
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", ErrEmptyContent
	}
	if len(text) > maxTextBytes {
		return "", ErrTextTooLarge
	}
	return text, nil
}

// uidHash is the only user identifier sent to the worker/Anthropic (ARCH §11.4).
func uidHash(userID uuid.UUID, pepper string) string {
	sum := sha256.Sum256([]byte(userID.String() + pepper))
	return hex.EncodeToString(sum[:])
}

// buildLLMMessages converts stored messages (+ their blocks) into the neutral
// llm.Message history. Pure. Only completed assistant turns and user turns with
// text are included; pending/failed/cancelled assistant turns are skipped.
func buildLLMMessages(msgs []db.Message, blocksByMsg map[uuid.UUID][]db.MessageBlock) []llm.Message {
	out := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "assistant" && m.Status != "complete" {
			continue
		}
		var content []llm.MessageBlock
		for _, blk := range blocksByMsg[m.ID] {
			if blk.Type == "text" && blk.ContentText.String != "" {
				content = append(content, llm.MessageBlock{Type: "text", Text: blk.ContentText.String})
			}
		}
		if len(content) == 0 {
			continue
		}
		out = append(out, llm.Message{Role: m.Role, Content: content})
	}
	return out
}

func toMessageView(m db.Message, blocks []db.MessageBlock) MessageView {
	content := make([]BlockView, 0, len(blocks))
	for _, b := range blocks {
		if b.Type == "text" {
			content = append(content, BlockView{Type: "text", Text: b.ContentText.String})
		}
	}
	return MessageView{
		ID:        m.ID.String(),
		Role:      m.Role,
		Status:    m.Status,
		CreatedAt: m.CreatedAt.Time,
		Content:   content,
	}
}

// --- cursor (keyset pagination over (created_at, id)) ---

func encodeCursor(t time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d:%s", t.UnixNano(), id.String())))
}

func decodeCursor(s string) (time.Time, uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, uuid.Nil, ErrBadCursor
	}
	nanoStr, idStr, ok := strings.Cut(string(raw), ":")
	if !ok {
		return time.Time{}, uuid.Nil, ErrBadCursor
	}
	nano, err := strconv.ParseInt(nanoStr, 10, 64)
	if err != nil {
		return time.Time{}, uuid.Nil, ErrBadCursor
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return time.Time{}, uuid.Nil, ErrBadCursor
	}
	return time.Unix(0, nano).UTC(), id, nil
}

func clampLimit(limit int) int32 {
	switch {
	case limit <= 0:
		return defaultPageLimit
	case limit > maxPageLimit:
		return maxPageLimit
	default:
		return int32(limit)
	}
}

// --- pgtype converters ---

func textVal(s string) pgtype.Text         { return pgtype.Text{String: s, Valid: s != ""} }
func tsVal(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

// int4 converts a token count to pgtype.Int4, clamping to the int32 range
// (token counts never approach it, but the conversion must be safe).
func int4(n int64) pgtype.Int4 {
	switch {
	case n < 0:
		n = 0
	case n > math.MaxInt32:
		n = math.MaxInt32
	}
	return pgtype.Int4{Int32: int32(n), Valid: true}
}
