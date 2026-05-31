package audio

import "context"

// GigaChatTranscriber is the reserved fallback provider (Sber SaluteSpeech /
// GigaAM). It is intentionally not implemented in v1: the codebase keeps the
// Transcriber seam open so SaluteSpeech can be dropped in later without touching
// callers (ROADMAP 4.1, ARCHITECTURE §11.5).
//
// Intended shape when implemented (credentials from env only):
//   - OAuth: POST https://ngw.devices.sberbank.ru:9443/api/v2/oauth
//     Authorization: Basic <GIGACHAT_API_KEY>, body scope=<GIGACHAT_SCOPE> ->
//     { "access_token": ..., "expires_at": ... }.
//   - Recognize: POST https://smartspeech.sber.ru/rest/v1/speech:recognize
//     Authorization: Bearer <access_token>, Content-Type: audio/ogg;codecs=opus,
//     body = OggOpus bytes -> { "result": ["..."], "status": 200 }.
type GigaChatTranscriber struct {
	apiKey string
	scope  string
}

func NewGigaChatTranscriber(cfg *Config) *GigaChatTranscriber {
	return &GigaChatTranscriber{apiKey: cfg.GigaChatAPIKey, scope: cfg.GigaChatScope}
}

var _ Transcriber = (*GigaChatTranscriber)(nil)

func (g *GigaChatTranscriber) Transcribe(context.Context, []byte, string) (Result, error) {
	return Result{}, ErrNotImplemented
}
