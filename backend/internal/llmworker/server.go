package llmworker

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/observability"
)

// Server wraps the HTTP routes around a provider (real Claude or echo).
type Server struct {
	logger   *slog.Logger
	cfg      *Config
	provider provider
}

// New selects the provider: real Anthropic when an API key is set, otherwise
// the echo fallback (dev only; prod requires the key — see LoadConfig).
func New(cfg *Config, logger *slog.Logger) *Server {
	var p provider
	if cfg.AnthropicAPIKey != "" {
		p = newAnthropicProvider(cfg.AnthropicAPIKey, cfg.MaxTokens, cfg.ModelOverride, logger)
	} else {
		logger.Warn("ANTHROPIC_API_KEY empty — using echo provider (dev only)")
		p = newEchoProvider()
	}
	return &Server{cfg: cfg, logger: logger, provider: p}
}

// Routes возвращает chi-роутер с применёнными middleware. Выделено отдельно,
// чтобы тесты могли поднять server через httptest.NewServer(srv.Routes()).
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(observability.AccessLog(s.logger))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Post("/v1/llm/messages", s.handleMessages)
	return r
}

// handleMessages validates the §11.3 payload (incl. PII allowlist) then streams
// the chosen provider's events as SSE. The provider owns the model call; this
// handler stays transport-only.
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB (text-only in 2.2)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "validation_failed", "read body: "+err.Error())
		return
	}
	defer r.Body.Close()

	var req messageRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json: "+err.Error())
		return
	}
	if err := req.validate(bodyBytes); err != nil {
		s.writeError(w, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	setSSEHeaders(w)
	w.WriteHeader(http.StatusOK)

	if err := s.provider.stream(r.Context(), req, &sseSink{w: w}); err != nil {
		// Client disconnect or stream write failure — connection is already
		// gone or being torn down; nothing more to send.
		s.logger.InfoContext(r.Context(), "stream ended early", "err", err.Error())
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": msg},
	})
}
