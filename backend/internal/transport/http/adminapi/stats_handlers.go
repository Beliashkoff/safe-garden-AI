package adminapi

import (
	"net/http"
	"strconv"
	"time"
)

func queryInt(r *http.Request, name string, def int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

// getOverview — GET /admin/v1/stats/overview.
func (h *Handler) getOverview(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.GetOverview(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"users_total":    o.UsersTotal,
		"users_new_7d":   o.UsersNew7d,
		"messages_total": o.MessagesTotal,
		"messages_7d":    o.Messages7d,
		"tokens_in_30d":  o.TokensIn30d,
		"tokens_out_30d": o.TokensOut30d,
		"cost_usd_30d":   o.CostUSD30d,
		"taps_30d":       o.Taps30d,
		"catalog_total":  o.CatalogTotal,
		"catalog_active": o.CatalogActive,
		"errors_24h":     o.Errors24h,

		"dau": o.DAU,
		"wau": o.WAU,
		"mau": o.MAU,

		"feedback_up_30d":       o.FeedbackUp30d,
		"feedback_down_30d":     o.FeedbackDown30d,
		"feedback_coverage_30d": o.FeedbackCoverage30d,

		"answers_complete_7d":  o.AnswersComplete7d,
		"answers_failed_7d":    o.AnswersFailed7d,
		"answers_cancelled_7d": o.AnswersCancelled7d,

		"accounts_deleted_total":   o.AccountsDeletedTotal,
		"media_purge_pending":      o.MediaPurgePending,
		"media_purge_oldest_hours": o.MediaPurgeOldestHours,

		"cost_mtd":            o.CostMTD,
		"cost_forecast_month": o.CostForecastMonth,
		"cost_prev_month":     o.CostPrevMonth,
	})
}

// getTimeseries — GET /admin/v1/stats/timeseries?days=30.
func (h *Handler) getTimeseries(w http.ResponseWriter, r *http.Request) {
	points, err := h.svc.GetTimeseries(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type pointDTO struct {
		Day       string  `json:"day"`
		Users     int64   `json:"users"`
		Messages  int64   `json:"messages"`
		TokensIn  int64   `json:"tokens_in"`
		TokensOut int64   `json:"tokens_out"`
		CostUSD   float64 `json:"cost_usd"`
	}
	out := make([]pointDTO, 0, len(points))
	for _, p := range points {
		out = append(out, pointDTO{
			Day:       p.Day.Format("2006-01-02"),
			Users:     p.Users,
			Messages:  p.Messages,
			TokensIn:  p.TokensIn,
			TokensOut: p.TokensOut,
			CostUSD:   p.CostUSD,
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"points": out})
}

// getTopProducts — GET /admin/v1/stats/top-products?days=30.
func (h *Handler) getTopProducts(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetTopProducts(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type tapDTO struct {
		Slug string `json:"slug"`
		Taps int64  `json:"taps"`
	}
	out := make([]tapDTO, 0, len(stats))
	for _, s := range stats {
		out = append(out, tapDTO{Slug: s.Slug, Taps: s.Taps})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"products": out})
}

// getErrors — GET /admin/v1/errors?limit=50&offset=0.
func (h *Handler) getErrors(w http.ResponseWriter, r *http.Request) {
	views, err := h.svc.ListErrors(r.Context(),
		int32(queryInt(r, "limit", 50)), int32(queryInt(r, "offset", 0))) //nolint:gosec // clamped in usecase
	if err != nil {
		respondError(w, r, err)
		return
	}
	type errorDTO struct {
		ID        int64  `json:"id"`
		Source    string `json:"source"`
		Route     string `json:"route"`
		Method    string `json:"method"`
		Status    int32  `json:"status"`
		RequestID string `json:"request_id,omitempty"`
		Message   string `json:"message,omitempty"`
		CreatedAt string `json:"created_at"`
	}
	out := make([]errorDTO, 0, len(views))
	for _, v := range views {
		out = append(out, errorDTO{
			ID:        v.ID,
			Source:    v.Source,
			Route:     v.Route,
			Method:    v.Method,
			Status:    v.Status,
			RequestID: v.RequestID,
			Message:   v.Message,
			CreatedAt: v.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"errors": out})
}

// getAudit — GET /admin/v1/audit?limit=50&offset=0.
func (h *Handler) getAudit(w http.ResponseWriter, r *http.Request) {
	views, err := h.svc.ListAudit(r.Context(),
		int32(queryInt(r, "limit", 50)), int32(queryInt(r, "offset", 0))) //nolint:gosec // clamped in usecase
	if err != nil {
		respondError(w, r, err)
		return
	}
	type auditDTO struct {
		ID        int64  `json:"id"`
		Action    string `json:"action"`
		Entity    string `json:"entity,omitempty"`
		EntityID  string `json:"entity_id,omitempty"`
		IP        string `json:"ip,omitempty"`
		CreatedAt string `json:"created_at"`
	}
	out := make([]auditDTO, 0, len(views))
	for _, v := range views {
		out = append(out, auditDTO{
			ID:        v.ID,
			Action:    v.Action,
			Entity:    v.Entity,
			EntityID:  v.EntityID,
			IP:        v.IP,
			CreatedAt: v.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"audit": out})
}

// getFeedback — GET /admin/v1/stats/feedback?days=30.
func (h *Handler) getFeedback(w http.ResponseWriter, r *http.Request) {
	points, err := h.svc.GetFeedbackSeries(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type pointDTO struct {
		Day  string `json:"day"`
		Up   int64  `json:"up"`
		Down int64  `json:"down"`
	}
	out := make([]pointDTO, 0, len(points))
	for _, p := range points {
		out = append(out, pointDTO{Day: p.Day.Format("2006-01-02"), Up: p.Up, Down: p.Down})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"points": out})
}

// getMessageStatus — GET /admin/v1/stats/message-status?days=30.
func (h *Handler) getMessageStatus(w http.ResponseWriter, r *http.Request) {
	points, err := h.svc.GetMessageStatusSeries(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type pointDTO struct {
		Day       string `json:"day"`
		Complete  int64  `json:"complete"`
		Failed    int64  `json:"failed"`
		Cancelled int64  `json:"cancelled"`
	}
	out := make([]pointDTO, 0, len(points))
	for _, p := range points {
		out = append(out, pointDTO{
			Day:       p.Day.Format("2006-01-02"),
			Complete:  p.Complete,
			Failed:    p.Failed,
			Cancelled: p.Cancelled,
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"points": out})
}

// getLoginBreakdown — GET /admin/v1/stats/login-breakdown?days=30.
func (h *Handler) getLoginBreakdown(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetLoginBreakdown(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type providerDTO struct {
		Provider string `json:"provider"`
		Logins   int64  `json:"logins"`
		Users    int64  `json:"users"`
	}
	out := make([]providerDTO, 0, len(stats))
	for _, s := range stats {
		out = append(out, providerDTO{Provider: s.Provider, Logins: s.Logins, Users: s.Users})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"providers": out})
}

// getTopUsers — GET /admin/v1/stats/top-users?days=7.
func (h *Handler) getTopUsers(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetTopCostUsers(r.Context(), queryInt(r, "days", 7))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type userDTO struct {
		User      string  `json:"user"`
		Requests  int64   `json:"requests"`
		TokensIn  int64   `json:"tokens_in"`
		TokensOut int64   `json:"tokens_out"`
		CostUSD   float64 `json:"cost_usd"`
	}
	out := make([]userDTO, 0, len(stats))
	for _, s := range stats {
		out = append(out, userDTO{
			User:      s.User,
			Requests:  s.Requests,
			TokensIn:  s.TokensIn,
			TokensOut: s.TokensOut,
			CostUSD:   s.CostUSD,
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"users": out})
}

// getErrorsBreakdown — GET /admin/v1/stats/errors-breakdown?days=7.
func (h *Handler) getErrorsBreakdown(w http.ResponseWriter, r *http.Request) {
	b, err := h.svc.GetErrorBreakdown(r.Context(), queryInt(r, "days", 7))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type routeDTO struct {
		Route  string `json:"route"`
		Status int32  `json:"status"`
		Count  int64  `json:"count"`
	}
	type dayDTO struct {
		Day   string `json:"day"`
		Count int64  `json:"count"`
	}
	routes := make([]routeDTO, 0, len(b.ByRoute))
	for _, rr := range b.ByRoute {
		routes = append(routes, routeDTO{Route: rr.Route, Status: rr.Status, Count: rr.Count})
	}
	days := make([]dayDTO, 0, len(b.ByDay))
	for _, d := range b.ByDay {
		days = append(days, dayDTO{Day: d.Day.Format("2006-01-02"), Count: d.Count})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"by_route": routes, "by_day": days})
}
