package chat

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"net/http"
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
	maxTextBytes        = 32 * 1024 // ARCH §8.2 text body cap
	historyLimit        = 20        // ARCH §7.4 context window
	finalizeTimeout     = 15 * time.Second
	defaultPageLimit    = 50
	maxPageLimit        = 100
	maxImagesPerMessage = 4        // images allowed in one user message
	maxImagesPerRequest = 4        // images hydrated (base64) into one model request
	imagePlaceholder    = "[фото]" // marker for history images beyond the cap
)

// imageContentTypes is the whitelist accepted for image_ref blocks (ARCH §8.2).
var imageContentTypes = map[string]struct{}{
	"image/jpeg": {}, "image/png": {}, "image/webp": {}, "image/heic": {}, "image/heif": {},
}

// messageLimiter is the per-user rate gate (consumer-side interface; satisfied
// by ratelimit.RedisLimiter / NoopMessage).
type messageLimiter interface {
	AllowMessage(ctx context.Context, userID uuid.UUID) (bool, error)
}

// imageStore reads uploaded objects back from storage (consumer-side interface;
// satisfied by *objstore.Client).
type imageStore interface {
	Get(ctx context.Context, key string) (data []byte, contentType string, err error)
}

// imageConverter detects HEIC and normalizes it to JPEG, which Claude (unlike
// HEIC) accepts. Satisfied by imageconv.Converter.
type imageConverter interface {
	IsHEIC(contentType string, data []byte) bool
	ToJPEG(data []byte) ([]byte, error)
}

// Service orchestrates chat persistence + the LLM client.
type Service struct {
	store   *storage.Store
	llm     llm.Client
	limiter messageLimiter
	images  imageStore
	conv    imageConverter
	pepper  string
	model   string
	logger  *slog.Logger
	now     func() time.Time
}

func NewService(
	store *storage.Store,
	client llm.Client,
	limiter messageLimiter,
	images imageStore,
	conv imageConverter,
	pepper, model string,
	logger *slog.Logger,
) *Service {
	return &Service{
		store:   store,
		llm:     client,
		limiter: limiter,
		images:  images,
		conv:    conv,
		pepper:  pepper,
		model:   model,
		logger:  logger,
		now:     time.Now,
	}
}

// validatedBlock is one accepted input block (kind: "text" | "image").
type validatedBlock struct {
	kind       string
	text       string
	storageKey string
}

// validateInput checks the request blocks (stage 3.1: text + image_ref) and
// returns them in order. Pure: the storage-key prefix is checked here, but DB
// ownership (GetUploadByStorageKey) is verified in SendMessage.
func validateInput(userID uuid.UUID, in SendInput) ([]validatedBlock, error) {
	if len(in.Blocks) == 0 {
		return nil, ErrEmptyContent
	}
	prefix := "u/" + userID.String() + "/"
	out := make([]validatedBlock, 0, len(in.Blocks))
	var textLen, images int
	hasText := false
	for _, blk := range in.Blocks {
		switch blk.Type {
		case "text":
			textLen += len(blk.Text)
			if strings.TrimSpace(blk.Text) != "" {
				hasText = true
			}
			out = append(out, validatedBlock{kind: "text", text: blk.Text})
		case "image_ref":
			if blk.StorageKey == "" || !strings.HasPrefix(blk.StorageKey, prefix) {
				return nil, ErrUploadNotFound
			}
			images++
			out = append(out, validatedBlock{kind: "image", storageKey: blk.StorageKey})
		default:
			return nil, ErrUnsupportedBlock
		}
	}
	if textLen > maxTextBytes {
		return nil, ErrTextTooLarge
	}
	if images > maxImagesPerMessage {
		return nil, ErrUnsupportedBlock
	}
	if !hasText && images == 0 {
		return nil, ErrEmptyContent
	}
	return out, nil
}

// uidHash is the only user identifier sent to the worker/Anthropic (ARCH §11.4).
func uidHash(userID uuid.UUID, pepper string) string {
	sum := sha256.Sum256([]byte(userID.String() + pepper))
	return hex.EncodeToString(sum[:])
}

// assembleMessages converts stored messages (+ their blocks) into the neutral
// llm.Message history. Pure. Completed assistant turns and user turns are kept;
// pending/failed/cancelled assistant turns are skipped. Image blocks whose key
// is in hydrated become base64 image blocks; other images degrade to a text
// placeholder so the model still knows a photo was sent.
func assembleMessages(
	msgs []db.Message,
	blocksByMsg map[uuid.UUID][]db.MessageBlock,
	hydrated map[string]llm.MessageBlock,
) []llm.Message {
	out := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "assistant" && m.Status != "complete" {
			continue
		}
		var content []llm.MessageBlock
		for _, blk := range blocksByMsg[m.ID] {
			switch blk.Type {
			case "text":
				if blk.ContentText.String != "" {
					content = append(content, llm.MessageBlock{Type: "text", Text: blk.ContentText.String})
				}
			case "image":
				key := blk.StorageKey.String
				if hb, ok := hydrated[key]; ok {
					content = append(content, hb)
				} else if key != "" {
					content = append(content, llm.MessageBlock{Type: "text", Text: imagePlaceholder})
				}
			}
		}
		if len(content) == 0 {
			continue
		}
		out = append(out, llm.Message{Role: m.Role, Content: content})
	}
	return out
}

// selectRecentImages returns up to n image storage keys, newest first, deduped.
// Pure. These are the only history images hydrated (base64) into the request;
// older ones become placeholders (bounds token cost — ARCH §13).
func selectRecentImages(msgs []db.Message, blocksByMsg map[uuid.UUID][]db.MessageBlock, n int) []string {
	keys := make([]string, 0, n)
	seen := make(map[string]struct{})
	for i := len(msgs) - 1; i >= 0 && len(keys) < n; i-- {
		m := msgs[i]
		if m.Role == "assistant" && m.Status != "complete" {
			continue
		}
		for _, blk := range blocksByMsg[m.ID] {
			if blk.Type != "image" || blk.StorageKey.String == "" {
				continue
			}
			key := blk.StorageKey.String
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
			if len(keys) >= n {
				break
			}
		}
	}
	return keys
}

// buildLLMHistory assembles the neutral history, hydrating the most recent
// images from object storage into base64 blocks (HEIC→JPEG as needed). A failed
// hydration degrades to a placeholder rather than failing the turn.
func (s *Service) buildLLMHistory(ctx context.Context, msgs []db.Message, blocksByMsg map[uuid.UUID][]db.MessageBlock) []llm.Message {
	keys := selectRecentImages(msgs, blocksByMsg, maxImagesPerRequest)
	hydrated := make(map[string]llm.MessageBlock, len(keys))
	for _, key := range keys {
		if block, ok := s.hydrateImage(ctx, key); ok {
			hydrated[key] = block
		}
	}
	return assembleMessages(msgs, blocksByMsg, hydrated)
}

// hydrateImage loads an image from object storage and returns it as a base64
// image block, converting HEIC→JPEG. Returns ok=false (logged) on any failure
// so the caller can fall back to a placeholder.
func (s *Service) hydrateImage(ctx context.Context, key string) (llm.MessageBlock, bool) {
	data, contentType, err := s.images.Get(ctx, key)
	if err != nil {
		s.logger.ErrorContext(ctx, "chat: image fetch failed", "err", err.Error())
		return llm.MessageBlock{}, false
	}
	if s.conv.IsHEIC(contentType, data) {
		jpg, err := s.conv.ToJPEG(data)
		if err != nil {
			s.logger.ErrorContext(ctx, "chat: heic convert failed", "err", err.Error())
			return llm.MessageBlock{}, false
		}
		data, contentType = jpg, "image/jpeg"
	}
	if !isClaudeImageType(contentType) {
		contentType = http.DetectContentType(data)
	}
	if !isClaudeImageType(contentType) {
		s.logger.WarnContext(ctx, "chat: unsupported image type for model", "content_type", contentType)
		return llm.MessageBlock{}, false
	}
	return llm.MessageBlock{
		Type:      "image",
		MediaB64:  base64.StdEncoding.EncodeToString(data),
		MediaType: contentType,
	}, true
}

// isClaudeImageType reports whether Anthropic accepts the media type directly.
func isClaudeImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func toMessageView(m db.Message, blocks []db.MessageBlock) MessageView {
	content := make([]BlockView, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case "text":
			content = append(content, BlockView{Type: "text", Text: b.ContentText.String})
		case "image":
			content = append(content, BlockView{Type: "image", StorageKey: b.StorageKey.String})
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
