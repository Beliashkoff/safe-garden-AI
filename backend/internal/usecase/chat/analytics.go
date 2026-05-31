package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// maxSlugLen bounds the slug recorded in usage_log.endpoint (defensive — slugs
// are short catalog identifiers, never free text).
const maxSlugLen = 128

// RecordFertilizerTap logs a tap on a fertilizer card for internal analytics
// (ROADMAP §5.3). It reuses usage_log with a synthetic endpoint
// "fertilizer_tap:<slug>" and zero tokens, so no schema change is needed. The
// slug is non-PII catalog data; nothing about message content is stored.
func (s *Service) RecordFertilizerTap(ctx context.Context, userID uuid.UUID, slug string) error {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ErrEmptyContent
	}
	if len(slug) > maxSlugLen {
		slug = slug[:maxSlugLen]
	}
	if err := s.store.InsertUsage(ctx, db.InsertUsageParams{
		UserID:    userID,
		Endpoint:  "fertilizer_tap:" + slug,
		TokensIn:  int4(0),
		TokensOut: int4(0),
	}); err != nil {
		return fmt.Errorf("chat: record fertilizer tap: %w", err)
	}
	return nil
}
