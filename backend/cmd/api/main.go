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

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/config"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/mailer"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/observability"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/ratelimit"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage"
	httptransport "github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/handler"
	authuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/auth"
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

	issuer, err := authpkg.NewIssuer(authpkg.IssuerConfig{
		KeysDir:        cfg.JWTKeysDir,
		ActiveKID:      cfg.JWTActiveKID,
		PrivateKeyPath: cfg.JWTPrivateKeyPath,
		KID:            cfg.JWTKID,
		AccessTTL:      cfg.JWTAccessTTL,
	})
	if err != nil {
		slog.Error("jwt issuer init failed", "err", err)
		os.Exit(1)
	}

	verifier, err := authpkg.NewVerifier(ctx, authpkg.VerifierConfig{
		AppleBundleID:    cfg.AppleBundleID,
		GoogleClientIOS:  cfg.GoogleClientIOS,
		GoogleClientAndr: cfg.GoogleClientAndr,
		GoogleClientWeb:  cfg.GoogleClientWeb,
	})
	if err != nil {
		slog.Error("oidc verifier init failed", "err", err)
		os.Exit(1)
	}

	mailerImpl := mailer.New(mailer.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		FromName: cfg.SMTPFromName,
		UseTLS:   cfg.SMTPTLS,
	}, logger)

	authService := authuc.NewService(
		store, issuer, verifier, mailerImpl,
		ratelimit.NewDB(store), cfg.RefreshTTL, logger,
	)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(observability.AccessLog(logger))

	r.Mount("/v1", httptransport.NewRouter(httptransport.Deps{
		Handler:     handler.New(authService),
		TokenParser: issuer,
		DocsEnabled: cfg.DocsEnabled,
	}))

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
