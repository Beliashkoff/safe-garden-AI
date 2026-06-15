package adminapi

import (
	"net/http"
	"time"
)

// --- Growth analytics ---

// getActivation — GET /admin/v1/stats/activation?days=30.
func (h *Handler) getActivation(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.GetActivation(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"signups":      a.Signups,
		"activated":    a.Activated,
		"median_hours": a.MedianHours,
	})
}

// getRetention — GET /admin/v1/stats/retention?days=84.
func (h *Handler) getRetention(w http.ResponseWriter, r *http.Request) {
	cohorts, err := h.svc.GetRetention(r.Context(), queryInt(r, "days", 84))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type cohortDTO struct {
		Week        string `json:"week"`
		Size        int64  `json:"size"`
		D1Eligible  int64  `json:"d1_eligible"`
		D1Retained  int64  `json:"d1_retained"`
		D7Eligible  int64  `json:"d7_eligible"`
		D7Retained  int64  `json:"d7_retained"`
		D30Eligible int64  `json:"d30_eligible"`
		D30Retained int64  `json:"d30_retained"`
	}
	out := make([]cohortDTO, 0, len(cohorts))
	for _, c := range cohorts {
		out = append(out, cohortDTO{
			Week:        c.Week.Format("2006-01-02"),
			Size:        c.Size,
			D1Eligible:  c.D1Eligible,
			D1Retained:  c.D1Retained,
			D7Eligible:  c.D7Eligible,
			D7Retained:  c.D7Retained,
			D30Eligible: c.D30Eligible,
			D30Retained: c.D30Retained,
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"cohorts": out})
}

// getConversationDepth — GET /admin/v1/stats/conversation-depth?days=30.
func (h *Handler) getConversationDepth(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.GetConversationDepth(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"bucket_1":   d.Bucket1,
		"bucket_2_4": d.Bucket2To4,
		"bucket_5_9": d.Bucket5To9,
		"bucket_10":  d.Bucket10,
		"avg":        d.Avg,
		"median":     d.Median,
		"p90":        d.P90,
	})
}

// getHeatmap — GET /admin/v1/stats/heatmap?days=30.
func (h *Handler) getHeatmap(w http.ResponseWriter, r *http.Request) {
	cells, err := h.svc.GetActivityHeatmap(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type cellDTO struct {
		DOW   int32 `json:"dow"`
		Hour  int32 `json:"hour"`
		Count int64 `json:"count"`
	}
	out := make([]cellDTO, 0, len(cells))
	for _, c := range cells {
		out = append(out, cellDTO{DOW: c.DOW, Hour: c.Hour, Count: c.Count})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"cells": out})
}

// getActivityWeekly — GET /admin/v1/stats/activity-weekly?days=180.
func (h *Handler) getActivityWeekly(w http.ResponseWriter, r *http.Request) {
	points, err := h.svc.GetActivityByWeek(r.Context(), queryInt(r, "days", 180))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type pointDTO struct {
		Week        string `json:"week"`
		Messages    int64  `json:"messages"`
		ActiveUsers int64  `json:"active_users"`
	}
	out := make([]pointDTO, 0, len(points))
	for _, p := range points {
		out = append(out, pointDTO{
			Week:        p.Week.Format("2006-01-02"),
			Messages:    p.Messages,
			ActiveUsers: p.ActiveUsers,
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"points": out})
}

// --- Answer quality ---

// getCTR — GET /admin/v1/stats/ctr?days=30.
func (h *Handler) getCTR(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.GetCTR(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type slugDTO struct {
		Slug        string `json:"slug"`
		Impressions int64  `json:"impressions"`
		Taps        int64  `json:"taps"`
	}
	perSlug := make([]slugDTO, 0, len(o.PerSlug))
	for _, s := range o.PerSlug {
		perSlug = append(perSlug, slugDTO{Slug: s.Slug, Impressions: s.Impressions, Taps: s.Taps})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"cards":       o.Cards,
		"impressions": o.Impressions,
		"taps":        o.Taps,
		"per_slug":    perSlug,
	})
}

// getLengthVsVerdict — GET /admin/v1/stats/length-vs-verdict?days=30.
func (h *Handler) getLengthVsVerdict(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.GetLengthByVerdict(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"avg_up":   l.AvgUp,
		"avg_down": l.AvgDown,
		"avg_none": l.AvgNone,
		"n_up":     l.NUp,
		"n_down":   l.NDown,
		"n_none":   l.NNone,
	})
}

// getNegativeConversations — GET /admin/v1/stats/negative-conversations?days=30&min=2.
func (h *Handler) getNegativeConversations(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.GetNegativeConversations(r.Context(), queryInt(r, "days", 30), queryInt(r, "min", 2))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type itemDTO struct {
		Conversation string `json:"conversation"`
		Downs        int64  `json:"downs"`
		LastDown     string `json:"last_down"`
	}
	out := make([]itemDTO, 0, len(items))
	for _, c := range items {
		out = append(out, itemDTO{
			Conversation: c.Conversation,
			Downs:        c.Downs,
			LastDown:     c.LastDown.Format(time.RFC3339),
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": out})
}

// getFollowup — GET /admin/v1/stats/followup?days=30.
func (h *Handler) getFollowup(w http.ResponseWriter, r *http.Request) {
	f, err := h.svc.GetFollowupRate(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"answers":   f.Answers,
		"followups": f.Followups,
	})
}

// getDownvoted — GET /admin/v1/stats/downvoted?limit=50&offset=0. Returns message
// content for operator review; never logged (CLAUDE.md invariant #3).
func (h *Handler) getDownvoted(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListDownvoted(r.Context(),
		int32(queryInt(r, "limit", 50)), int32(queryInt(r, "offset", 0))) //nolint:gosec // clamped in usecase
	if err != nil {
		respondError(w, r, err)
		return
	}
	type itemDTO struct {
		Conversation string `json:"conversation"`
		Question     string `json:"question,omitempty"`
		Answer       string `json:"answer,omitempty"`
		HadCard      bool   `json:"had_card"`
		CreatedAt    string `json:"created_at"`
	}
	out := make([]itemDTO, 0, len(items))
	for _, m := range items {
		out = append(out, itemDTO{
			Conversation: m.Conversation,
			Question:     m.Question,
			Answer:       m.Answer,
			HadCard:      m.HadCard,
			CreatedAt:    m.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": out})
}
