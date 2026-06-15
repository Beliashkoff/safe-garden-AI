package adminapi

import "net/http"

// getUnitEconomics — GET /admin/v1/stats/unit-economics?days=30.
func (h *Handler) getUnitEconomics(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.GetUnitEconomics(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"cost_usd":         u.CostUSD,
		"messages":         u.Messages,
		"users":            u.Users,
		"tokens_in":        u.TokensIn,
		"tokens_out":       u.TokensOut,
		"cost_per_message": u.CostPerMessage,
		"cost_per_user":    u.CostPerUser,
	})
}

// getCacheStats — GET /admin/v1/stats/cache?days=30.
func (h *Handler) getCacheStats(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.GetCacheStats(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"input_uncached": c.InputUncached,
		"cache_write":    c.CacheWrite,
		"cache_read":     c.CacheRead,
		"hit_rate":       c.HitRate,
		"savings_usd":    c.SavingsUSD,
		"has_data":       c.HasData,
	})
}

// getCostByModel — GET /admin/v1/stats/cost-by-model?days=30.
func (h *Handler) getCostByModel(w http.ResponseWriter, r *http.Request) {
	models, err := h.svc.GetCostByModel(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type modelDTO struct {
		Model     string  `json:"model"`
		CostUSD   float64 `json:"cost_usd"`
		TokensIn  int64   `json:"tokens_in"`
		TokensOut int64   `json:"tokens_out"`
		Requests  int64   `json:"requests"`
	}
	out := make([]modelDTO, 0, len(models))
	for _, m := range models {
		out = append(out, modelDTO{
			Model:     m.Model,
			CostUSD:   m.CostUSD,
			TokensIn:  m.TokensIn,
			TokensOut: m.TokensOut,
			Requests:  m.Requests,
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"models": out})
}

// getCostByKind — GET /admin/v1/stats/cost-by-kind?days=30.
func (h *Handler) getCostByKind(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.GetCostByKind(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"claude_cost":    c.ClaudeCost,
		"claude_calls":   c.ClaudeCalls,
		"taps":           c.Taps,
		"transcriptions": c.Transcriptions,
		"transcribe_sec": c.TranscribeSec,
	})
}
