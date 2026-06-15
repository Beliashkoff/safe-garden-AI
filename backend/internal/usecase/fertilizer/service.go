// Package fertilizer resolves recommend_fertilizer tool calls against the
// catalog (ARCH §6.4). It is invoked by the internal mTLS endpoint that the
// llm-worker calls back into during a Claude tool-use turn.
package fertilizer

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// Store is the catalog query subset the service needs (defined here, on the
// consumer side, per the project's interface convention).
type Store interface {
	RecommendFertilizers(ctx context.Context, arg db.RecommendFertilizersParams) ([]db.RecommendFertilizersRow, error)
	InsertDiagEvent(ctx context.Context, arg db.InsertDiagEventParams) error
}

// Service selects catalog products for a diagnosed problem.
type Service struct {
	store Store
}

// NewService builds the fertilizer service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Recommend returns 0–3 catalog products matching the tool args (ARCH §6.4). An
// empty slice means no match — empty catalog (SPEC Q2) or nothing fits — and the
// caller then answers with text only, no card.
func (s *Service) Recommend(ctx context.Context, args llm.FertilizerToolArgs) ([]llm.FertilizerProduct, error) {
	if args.Problem == "" {
		return nil, fmt.Errorf("fertilizer: problem is required")
	}

	params := db.RecommendFertilizersParams{Problem: args.Problem}
	if args.Plant != "" {
		params.Plant = pgtype.Text{String: args.Plant, Valid: true}
	}

	rows, err := s.store.RecommendFertilizers(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("fertilizer: query: %w", err)
	}

	out := make([]llm.FertilizerProduct, 0, len(rows))
	for _, r := range rows {
		p := llm.FertilizerProduct{
			ID:          r.ID.String(),
			Slug:        r.Slug,
			Name:        r.Name,
			ShortDesc:   r.ShortDesc,
			ImageURL:    r.ImageUrl.String,    // zero value "" when NULL
			DeeplinkURL: r.DeeplinkUrl.String, // zero value "" when NULL
		}
		if r.PriceRub.Valid {
			price := r.PriceRub.Int32
			p.PriceRub = &price
		}
		out = append(out, p)
	}

	// Record the diagnosis for catalog analytics (problem distribution + miss-rate).
	// Best-effort, non-PII (problem is a closed enum); never fails the recommendation.
	_ = s.store.InsertDiagEvent(ctx, db.InsertDiagEventParams{Problem: args.Problem, Matched: len(out) > 0})

	return out, nil
}
