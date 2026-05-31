// Package internalapi serves the mTLS-only service endpoints the llm-worker
// calls back into during a tool-use turn (ARCH §11). It runs on a SEPARATE
// listener from the public /v1 API and is never behind RequireAuth — the client
// certificate is the authentication. No PII crosses this boundary: anonymized
// tool args go in, catalog products come out (CLAUDE.md invariant #5).
package internalapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/observability"
)

const maxBodyBytes = 1 << 16 // 64 KiB — tool args are tiny

// recommender resolves recommend_fertilizer args into catalog products.
type recommender interface {
	Recommend(ctx context.Context, args llm.FertilizerToolArgs) ([]llm.FertilizerProduct, error)
}

// Handler is the internal API surface.
type Handler struct {
	fert   recommender
	logger *slog.Logger
}

// New builds the internal handler.
func New(fert recommender, logger *slog.Logger) *Handler {
	return &Handler{fert: fert, logger: logger}
}

// Routes returns the internal router. Kept separate so it can be served on its
// own (mTLS) listener and unit-tested via httptest.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(observability.AccessLog(h.logger))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Post("/internal/v1/tools/fertilizer", h.PostFertilizer)
	return r
}

// PostFertilizer handles the recommend_fertilizer callback (ARCH §11):
// body { args: { problem, plant?, severity?, notes? } } → { products: [...] }.
func (h *Handler) PostFertilizer(w http.ResponseWriter, r *http.Request) {
	var req llm.FertilizerToolRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if req.Args.Problem == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "args.problem is required")
		return
	}

	products, err := h.fert.Recommend(r.Context(), req.Args)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "fertilizer recommend failed", "err", err.Error())
		writeError(w, http.StatusInternalServerError, "recommend_failed", "could not resolve recommendations")
		return
	}
	if products == nil {
		products = []llm.FertilizerProduct{}
	}
	writeJSON(w, http.StatusOK, llm.FertilizerToolResponse{Products: products})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
