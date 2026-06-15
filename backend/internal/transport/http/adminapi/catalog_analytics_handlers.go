package adminapi

import "net/http"

// getProblemDistribution — GET /admin/v1/stats/problems?days=30.
func (h *Handler) getProblemDistribution(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetProblemDistribution(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type problemDTO struct {
		Problem string `json:"problem"`
		Total   int64  `json:"total"`
		Misses  int64  `json:"misses"`
	}
	out := make([]problemDTO, 0, len(stats))
	for _, s := range stats {
		out = append(out, problemDTO{Problem: s.Problem, Total: s.Total, Misses: s.Misses})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"problems": out})
}

// getCatalogPerformance — GET /admin/v1/stats/catalog-performance?days=30.
func (h *Handler) getCatalogPerformance(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.GetCatalogPerformance(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type productDTO struct {
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		Impressions int64  `json:"impressions"`
		Taps        int64  `json:"taps"`
	}
	out := make([]productDTO, 0, len(products))
	for _, p := range products {
		out = append(out, productDTO{Slug: p.Slug, Name: p.Name, Impressions: p.Impressions, Taps: p.Taps})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"products": out})
}
