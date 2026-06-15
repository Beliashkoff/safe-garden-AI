package adminapi

import (
	"net/http"
	"time"
)

// getDeletionEvents — GET /admin/v1/compliance/deletions?limit=200&offset=0.
func (h *Handler) getDeletionEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.svc.ListDeletionEvents(r.Context(),
		int32(queryInt(r, "limit", 200)), int32(queryInt(r, "offset", 0))) //nolint:gosec // clamped in usecase
	if err != nil {
		respondError(w, r, err)
		return
	}
	type eventDTO struct {
		User      string `json:"user,omitempty"`
		Action    string `json:"action"`
		CreatedAt string `json:"created_at"`
	}
	out := make([]eventDTO, 0, len(events))
	for _, e := range events {
		out = append(out, eventDTO{User: e.User, Action: e.Action, CreatedAt: e.CreatedAt.Format(time.RFC3339)})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"events": out})
}

// getStalePurges — GET /admin/v1/compliance/stale-purges.
func (h *Handler) getStalePurges(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListStalePurges(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	type itemDTO struct {
		User         string  `json:"user"`
		DeletedAt    string  `json:"deleted_at"`
		PendingHours float64 `json:"pending_hours"`
	}
	out := make([]itemDTO, 0, len(items))
	for _, p := range items {
		out = append(out, itemDTO{
			User:         p.User,
			DeletedAt:    p.DeletedAt.Format(time.RFC3339),
			PendingHours: p.PendingHours,
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": out})
}

// getComplianceOverview — GET /admin/v1/compliance/overview. Aggregates the
// upload-GC, retention-backlog, usage-residue and cleanup-heartbeat numbers.
func (h *Handler) getComplianceOverview(w http.ResponseWriter, r *http.Request) {
	gc, err := h.svc.GetUploadGCStats(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	ret, err := h.svc.GetRetentionBacklog(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	res, err := h.svc.GetUsageResidue(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	cu, err := h.svc.GetCleanupHealth(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	cleanup := map[string]any{
		"recorded":      cu.Recorded,
		"stale":         cu.Stale,
		"users_purged":  cu.UsersPurged,
		"uploads_gc":    cu.UploadsGC,
		"admin_rows_gc": cu.AdminRowsGC,
	}
	if cu.Recorded {
		cleanup["last_run_at"] = cu.LastRunAt.Format(time.RFC3339)
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"uploads": map[string]any{
			"unused_total": gc.UnusedTotal,
			"stale_total":  gc.StaleTotal,
			"stale_bytes":  gc.StaleBytes,
		},
		"retention": map[string]any{
			"expired_otp":   ret.ExpiredOTP,
			"expired_oauth": ret.ExpiredOAuth,
			"old_revoked":   ret.OldRevoked,
		},
		"usage_residue": map[string]any{
			"rows":         res.Rows,
			"users":        res.Users,
			"oldest_hours": res.OldestHours,
			"has_residue":  res.HasResidue,
		},
		"cleanup": cleanup,
	})
}
