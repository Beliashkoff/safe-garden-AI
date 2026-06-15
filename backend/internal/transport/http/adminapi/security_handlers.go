package adminapi

import (
	"net/http"
	"time"
)

// getSecurityEvents — GET /admin/v1/security/events?limit=50&offset=0.
func (h *Handler) getSecurityEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.svc.ListSecurityEvents(r.Context(),
		int32(queryInt(r, "limit", 50)), int32(queryInt(r, "offset", 0))) //nolint:gosec // clamped in usecase
	if err != nil {
		respondError(w, r, err)
		return
	}
	type eventDTO struct {
		User      string `json:"user,omitempty"`
		Action    string `json:"action"`
		IP        string `json:"ip,omitempty"`
		CreatedAt string `json:"created_at"`
	}
	out := make([]eventDTO, 0, len(events))
	for _, e := range events {
		out = append(out, eventDTO{
			User:      e.User,
			Action:    e.Action,
			IP:        e.IP,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"events": out})
}

// getOtpStats — GET /admin/v1/security/otp?days=7.
func (h *Handler) getOtpStats(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.GetOtpStats(r.Context(), queryInt(r, "days", 7))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type requesterDTO struct {
		Email string `json:"email"`
		Codes int64  `json:"codes"`
	}
	reqs := make([]requesterDTO, 0, len(o.TopRequesters))
	for _, rq := range o.TopRequesters {
		reqs = append(reqs, requesterDTO{Email: rq.Email, Codes: rq.Codes})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"issued":         o.Issued,
		"used":           o.Used,
		"exhausted":      o.Exhausted,
		"expired_unused": o.ExpiredUnused,
		"delivery_rate":  o.DeliveryRate,
		"top_requesters": reqs,
	})
}

// getSuspiciousIPs — GET /admin/v1/security/suspicious-ips?days=7.
func (h *Handler) getSuspiciousIPs(w http.ResponseWriter, r *http.Request) {
	ips, err := h.svc.GetSuspiciousIPs(r.Context(), queryInt(r, "days", 7))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type ipDTO struct {
		Source string `json:"source"`
		IP     string `json:"ip"`
		Count  int64  `json:"count"`
	}
	out := make([]ipDTO, 0, len(ips))
	for _, i := range ips {
		out = append(out, ipDTO{Source: i.Source, IP: i.IP, Count: i.Count})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"ips": out})
}
