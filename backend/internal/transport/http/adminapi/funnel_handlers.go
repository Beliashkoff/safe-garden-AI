package adminapi

import "net/http"

// getInputFunnel — GET /admin/v1/stats/input-funnel?days=30.
func (h *Handler) getInputFunnel(w http.ResponseWriter, r *http.Request) {
	f, err := h.svc.GetInputFunnel(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"total":          f.Total,
		"photo":          f.Photo,
		"voice":          f.Voice,
		"photo_answered": f.PhotoAnswered,
		"photo_ok":       f.PhotoOK,
		"voice_answered": f.VoiceAnswered,
		"voice_ok":       f.VoiceOK,
		"transcriptions": f.Transcriptions,
		"transcribe_sec": f.TranscribeSec,
	})
}

// getInputByDay — GET /admin/v1/stats/input-by-day?days=30.
func (h *Handler) getInputByDay(w http.ResponseWriter, r *http.Request) {
	points, err := h.svc.GetInputTypeSeries(r.Context(), queryInt(r, "days", 30))
	if err != nil {
		respondError(w, r, err)
		return
	}
	type pointDTO struct {
		Day   string `json:"day"`
		Photo int64  `json:"photo"`
		Voice int64  `json:"voice"`
		Text  int64  `json:"text"`
	}
	out := make([]pointDTO, 0, len(points))
	for _, p := range points {
		out = append(out, pointDTO{Day: p.Day.Format("2006-01-02"), Photo: p.Photo, Voice: p.Voice, Text: p.Text})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"points": out})
}
