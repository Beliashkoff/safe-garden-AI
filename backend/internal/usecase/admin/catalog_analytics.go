package admin

import (
	"context"
	"fmt"
)

// Catalog & assortment analytics for the "Ассортимент" admin page: which plant
// problems the model diagnoses (and which the catalog cannot answer), plus
// per-product impressions/taps including dead inventory.

// ProblemStat is one diagnosed problem with its catalog miss count.
type ProblemStat struct {
	Problem string
	Total   int64
	Misses  int64
}

// GetProblemDistribution returns diagnosed-problem frequency + miss-rate for the
// last `days`. Empty until diag_events accumulates after the deploy.
func (s *Service) GetProblemDistribution(ctx context.Context, days int) ([]ProblemStat, error) {
	rows, err := s.store.ProblemDistributionSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return nil, fmt.Errorf("admin: problem distribution: %w", err)
	}
	out := make([]ProblemStat, 0, len(rows))
	for _, r := range rows {
		out = append(out, ProblemStat{Problem: r.Problem, Total: r.Total, Misses: r.Misses})
	}
	return out, nil
}

// CatalogProduct is one active catalog item with its window activity.
type CatalogProduct struct {
	Slug        string
	Name        string
	Impressions int64
	Taps        int64
}

// GetCatalogPerformance returns every active product with impressions/taps for
// the last `days`. Products with zero of both are dead assortment.
func (s *Service) GetCatalogPerformance(ctx context.Context, days int) ([]CatalogProduct, error) {
	rows, err := s.store.CatalogPerformanceSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return nil, fmt.Errorf("admin: catalog performance: %w", err)
	}
	out := make([]CatalogProduct, 0, len(rows))
	for _, r := range rows {
		out = append(out, CatalogProduct{
			Slug:        r.Slug,
			Name:        r.Name,
			Impressions: r.Impressions,
			Taps:        r.Taps,
		})
	}
	return out, nil
}
