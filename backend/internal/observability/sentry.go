package observability

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
)

func InitSentry(dsn, env string) error {
	if dsn == "" {
		return nil
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		EnableTracing:    false,
		TracesSampleRate: 0,
	}); err != nil {
		return fmt.Errorf("sentry init: %w", err)
	}
	return nil
}

func FlushSentry() {
	sentry.Flush(2 * time.Second)
}
