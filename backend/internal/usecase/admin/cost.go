package admin

import (
	"context"
	"fmt"
	"sort"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
)

// FinOps analytics for the "Стоимость" admin page: unit economics, prompt-cache
// efficiency, cost by model and cost by operation kind. Cache savings are priced
// through the canonical llm.EstimateCostUSD so the rate lives in one place.

// UnitEconomics is cost-per-message and cost-per-active-user over a window.
type UnitEconomics struct {
	CostUSD        float64
	Messages       int64
	Users          int64
	TokensIn       int64
	TokensOut      int64
	CostPerMessage float64
	CostPerUser    float64
}

// GetUnitEconomics returns the unit-economics summary for the last `days`.
func (s *Service) GetUnitEconomics(ctx context.Context, days int) (UnitEconomics, error) {
	r, err := s.store.UnitEconomicsSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return UnitEconomics{}, fmt.Errorf("admin: unit economics: %w", err)
	}
	u := UnitEconomics{
		CostUSD:   numericToFloat(r.CostUsd),
		Messages:  r.Messages,
		Users:     r.Users,
		TokensIn:  r.TokensIn,
		TokensOut: r.TokensOut,
	}
	if u.Messages > 0 {
		u.CostPerMessage = u.CostUSD / float64(u.Messages)
	}
	if u.Users > 0 {
		u.CostPerUser = u.CostUSD / float64(u.Users)
	}
	return u, nil
}

// CacheStats is prompt-cache hit-rate and the dollars saved by cache reads.
type CacheStats struct {
	InputUncached int64
	CacheWrite    int64
	CacheRead     int64
	HitRate       float64
	SavingsUSD    float64
	HasData       bool
}

// GetCacheStats returns prompt-cache efficiency for the last `days`. Only rows
// written after migration 0017 carry the split, so HasData reflects whether any
// post-deploy data exists yet.
func (s *Service) GetCacheStats(ctx context.Context, days int) (CacheStats, error) {
	r, err := s.store.CacheStatsSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return CacheStats{}, fmt.Errorf("admin: cache stats: %w", err)
	}
	c := CacheStats{
		InputUncached: r.InputUncached,
		CacheWrite:    r.CacheWrite,
		CacheRead:     r.CacheRead,
		HasData:       r.RowsWithData > 0,
	}
	total := r.InputUncached + r.CacheWrite + r.CacheRead
	if total > 0 {
		c.HitRate = float64(r.CacheRead) / float64(total)
	}
	// Savings = what those cache-read tokens would have cost uncached minus what
	// they actually cost at the 0.1x cache-read rate (priced via the canonical model).
	if r.CacheRead > 0 {
		full := llm.EstimateCostUSD(llm.DefaultModel, llm.TokenUsage{InputTokens: r.CacheRead})
		cached := llm.EstimateCostUSD(llm.DefaultModel, llm.TokenUsage{CacheReadTokens: r.CacheRead})
		c.SavingsUSD = full - cached
	}
	return c, nil
}

// ModelCost is spend and tokens for one model id over a window.
type ModelCost struct {
	Model     string
	CostUSD   float64
	TokensIn  int64
	TokensOut int64
	Requests  int64
}

// GetCostByModel returns spend grouped by model id for the last `days`.
func (s *Service) GetCostByModel(ctx context.Context, days int) ([]ModelCost, error) {
	rows, err := s.store.CostByModelSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return nil, fmt.Errorf("admin: cost by model: %w", err)
	}
	out := make([]ModelCost, 0, len(rows))
	for _, r := range rows {
		out = append(out, ModelCost{
			Model:     r.Model,
			CostUSD:   numericToFloat(r.CostUsd),
			TokensIn:  r.TokensIn,
			TokensOut: r.TokensOut,
			Requests:  r.Requests,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CostUSD > out[j].CostUSD })
	return out, nil
}

// CostByKind splits activity by operation: paid Claude turns, free catalog taps,
// and voice transcriptions (SpeechKit, billed in rubles by duration).
type CostByKind struct {
	ClaudeCost     float64
	ClaudeCalls    int64
	Taps           int64
	Transcriptions int64
	TranscribeSec  int64
}

// GetCostByKind returns the per-operation breakdown for the last `days`.
func (s *Service) GetCostByKind(ctx context.Context, days int) (CostByKind, error) {
	r, err := s.store.CostByKindSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return CostByKind{}, fmt.Errorf("admin: cost by kind: %w", err)
	}
	return CostByKind{
		ClaudeCost:     numericToFloat(r.ClaudeCost),
		ClaudeCalls:    r.ClaudeCalls,
		Taps:           r.Taps,
		Transcriptions: r.Transcriptions,
		TranscribeSec:  r.TranscribeMs / 1000,
	}, nil
}
