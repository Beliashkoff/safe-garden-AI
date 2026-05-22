package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/config"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/observability"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.Env, cfg.LogLevel)
	slog.SetDefault(logger)

	if err := observability.InitSentry(cfg.SentryDSN, cfg.Env); err != nil {
		slog.Error("sentry init failed", "err", err)
	}
	defer observability.FlushSentry()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := storage.New(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("storage init failed", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(observability.AccessLog(logger))

	// Liveness — succeeds as long as the process is running.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// Readiness — pings the DB with a bounded timeout so a stalled DB does
	// not hang the probe and orchestrator decisions.
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		pingCtx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := store.Ping(pingCtx); err != nil {
			slog.Warn("readyz ping failed", "err", err)
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("server shutting down")

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("server shutdown error", "err", err)
	}
}
