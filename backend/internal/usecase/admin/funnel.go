package admin

import (
	"context"
	"fmt"
	"time"
)

// Photo & voice input funnel (product core): share of photo/voice/text questions
// and the answer success-rate of each path. Backs the funnel section of the
// "Рост" page.

// InputFunnel is the photo/voice/text breakdown plus per-path answer success.
type InputFunnel struct {
	Total          int64
	Photo          int64
	Voice          int64
	PhotoAnswered  int64
	PhotoOK        int64
	VoiceAnswered  int64
	VoiceOK        int64
	Transcriptions int64
	TranscribeSec  int64
}

// GetInputFunnel returns the photo/voice funnel for the last `days`.
func (s *Service) GetInputFunnel(ctx context.Context, days int) (InputFunnel, error) {
	since := s.sinceDays(days, 30)
	f, err := s.store.PhotoVoiceFunnelSince(ctx, since)
	if err != nil {
		return InputFunnel{}, fmt.Errorf("admin: input funnel: %w", err)
	}
	tr, err := s.store.TranscriptionVolumeSince(ctx, since)
	if err != nil {
		return InputFunnel{}, fmt.Errorf("admin: transcription volume: %w", err)
	}
	return InputFunnel{
		Total:          f.Total,
		Photo:          f.Photo,
		Voice:          f.Voice,
		PhotoAnswered:  f.PhotoAnswered,
		PhotoOK:        f.PhotoOk,
		VoiceAnswered:  f.VoiceAnswered,
		VoiceOK:        f.VoiceOk,
		Transcriptions: tr.Count,
		TranscribeSec:  tr.TotalMs / 1000,
	}, nil
}

// InputTypePoint is one day of the input-type mix.
type InputTypePoint struct {
	Day   time.Time
	Photo int64
	Voice int64
	Text  int64
}

// GetInputTypeSeries returns the daily photo/voice/text mix for the last `days`.
func (s *Service) GetInputTypeSeries(ctx context.Context, days int) ([]InputTypePoint, error) {
	rows, err := s.store.InputTypeByDaySince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return nil, fmt.Errorf("admin: input type by day: %w", err)
	}
	out := make([]InputTypePoint, 0, len(rows))
	for _, r := range rows {
		text := r.Total - r.Photo - r.Voice
		if text < 0 {
			text = 0
		}
		out = append(out, InputTypePoint{Day: r.Day.Time, Photo: r.Photo, Voice: r.Voice, Text: text})
	}
	return out, nil
}
