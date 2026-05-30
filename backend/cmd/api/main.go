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
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/config"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/imageconv"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/mailer"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/objstore"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/observability"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/ratelimit"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage"
	httptransport "github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/handler"
	authuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/auth"
	chatuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/chat"
	uploaduc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/upload"
)

// objStore is the subset of object-storage operations the usecases need
// (presigned uploads + server-side reads). Satisfied by *objstore.Client and
// objstore.Disabled.
type objStore interface {
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, map[string]string, error)
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
	Get(ctx context.Context, key string) ([]byte, string, error)
}

// buildObjStore returns the configured object store, or a Disabled stub when S3
// is unconfigured (text-only dev without MinIO). Fatal on a present-but-broken
// S3 config.
func buildObjStore(cfg *config.Config) objStore {
	if cfg.S3Bucket == "" {
		slog.Warn("S3_BUCKET empty — photo uploads disabled (dev only)")
		return objstore.Disabled{}
	}
	client, err := objstore.New(objstore.Config{
		Endpoint:     cfg.S3Endpoint,
		Region:       cfg.S3Region,
		AccessKey:    cfg.S3AccessKey,
		SecretKey:    cfg.S3SecretKey,
		Bucket:       cfg.S3Bucket,
		UsePathStyle: cfg.S3UsePathStyle,
	})
	if err != nil {
		slog.Error("object storage init failed", "err", err)
		os.Exit(1)
	}
	return client
}

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

	// LLM client: worker (mTLS+SSE) or mock, per LLM_CLIENT_KIND.
	llmCfg, err := llm.LoadConfig()
	if err != nil {
		slog.Error("llm config load failed", "err", err)
		os.Exit(1)
	}
	llmClient, err := llm.New(llmCfg)
	if err != nil {
		slog.Error("llm client init failed", "err", err)
		os.Exit(1)
	}

	// Per-user message rate limit (ARCH §8.2). Redis when configured; otherwise a
	// no-op allow-all for local dev without Redis.
	var msgLimiter interface {
		AllowMessage(ctx context.Context, userID uuid.UUID) (bool, error)
	}
	if cfg.RedisAddr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
		defer func() { _ = rdb.Close() }()
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		if err := rdb.Ping(pingCtx).Err(); err != nil {
			cancel()
			slog.Error("redis ping failed", "err", err)
			os.Exit(1)
		}
		cancel()
		msgLimiter = ratelimit.NewRedis(rdb, logger)
	} else {
		slog.Warn("REDIS_ADDR empty — message rate limiting disabled (dev only)")
		msgLimiter = ratelimit.NewNoopMessage()
	}

	// Object storage for presigned photo uploads (ARCH §4.3, §5).
	objs := buildObjStore(cfg)
	uploadService := uploaduc.NewService(store, objs)
	chatService := chatuc.NewService(
		store, llmClient, msgLimiter, objs, imageconv.New(),
		cfg.UIDHashPepper, llm.DefaultModel, logger,
	)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(observability.AccessLog(logger))

	r.Mount("/v1", httptransport.NewRouter(httptransport.Deps{
		Handler:     handler.New(authService, chatService, uploadService),
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
