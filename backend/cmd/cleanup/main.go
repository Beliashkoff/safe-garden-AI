// Command cleanup runs the media-purge + upload-GC jobs once and exits
// (ROADMAP §3.2). A scheduler (system cron or `docker compose run --rm cleanup`)
// owns the cadence; recommended hourly so deleted-account media is removed
// promptly and the 7-day unused-upload GC stays cheap. Idempotent — safe to
// re-run.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/config"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/objstore"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/observability"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/cleanup"
)

const runTimeout = 5 * time.Minute

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.Env, cfg.LogLevel)
	slog.SetDefault(logger)

	if cfg.S3Bucket == "" {
		slog.Warn("S3_BUCKET empty — object storage not configured, nothing to clean")
		os.Exit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	store, err := storage.New(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("storage init failed", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	objs, err := objstore.New(objstore.Config{
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

	if err := cleanup.NewService(store, objs, logger).RunOnce(ctx); err != nil {
		slog.Error("cleanup run failed", "err", err)
		os.Exit(1)
	}
}
