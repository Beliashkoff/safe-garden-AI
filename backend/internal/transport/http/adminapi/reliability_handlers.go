package adminapi

import (
	"net/http"
	"time"
)

// getWorkerHealth — GET /admin/v1/health/worker.
func (h *Handler) getWorkerHealth(w http.ResponseWriter, r *http.Request) {
	wh := h.svc.GetWorkerHealth(r.Context())
	body := map[string]any{
		"configured": wh.Configured,
		"online":     wh.Online,
		"latency_ms": wh.LatencyMS,
		"model":      wh.Model,
	}
	if !wh.CheckedAt.IsZero() {
		body["checked_at"] = wh.CheckedAt.Format(time.RFC3339)
	}
	writeJSON(w, r, http.StatusOK, body)
}

// getDependencies — GET /admin/v1/health/deps.
func (h *Handler) getDependencies(w http.ResponseWriter, r *http.Request) {
	deps := h.svc.GetDependencies(r.Context())
	type depDTO struct {
		Name       string `json:"name"`
		Configured bool   `json:"configured"`
		Up         bool   `json:"up"`
		LatencyMS  int64  `json:"latency_ms"`
		Detail     string `json:"detail,omitempty"`
	}
	out := make([]depDTO, 0, len(deps))
	for _, d := range deps {
		out = append(out, depDTO{
			Name:       d.Name,
			Configured: d.Configured,
			Up:         d.Up,
			LatencyMS:  d.LatencyMS,
			Detail:     d.Detail,
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"deps": out})
}

// getFailCodes — GET /admin/v1/stats/fail-codes?days=14.
func (h *Handler) getFailCodes(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetFailCodes(r.Context(), queryInt(r, "days", 14))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type codeDTO struct {
		Code  string `json:"code"`
		Count int64  `json:"count"`
	}
	out := make([]codeDTO, 0, len(stats))
	for _, s := range stats {
		out = append(out, codeDTO{Code: s.Code, Count: s.Count})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"codes": out})
}

// getAlerts — GET /admin/v1/alerts.
func (h *Handler) getAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.svc.EvaluateAlerts(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	type alertDTO struct {
		Level   string `json:"level"`
		Metric  string `json:"metric"`
		Message string `json:"message"`
	}
	out := make([]alertDTO, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, alertDTO{Level: a.Level, Metric: a.Metric, Message: a.Message})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"alerts": out})
}
