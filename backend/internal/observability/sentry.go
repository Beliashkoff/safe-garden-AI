package observability

import (
	"fmt"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
		// Never attach request body, headers, cookies, or client IP to events.
		SendDefaultPII: false,
		// Defense in depth (CLAUDE.md invariant #3): strip the request and user
		// objects from every event so no URL/query/header/cookie or email/id can
		// leak even if a future code path attaches one.
		BeforeSend: func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
			event.Request = nil
			event.User = sentry.User{}
			return event
		},
	}); err != nil {
		return fmt.Errorf("sentry init: %w", err)
	}
	return nil
}

func FlushSentry() {
	sentry.Flush(2 * time.Second)
}

// SentryMiddleware reports panics (with a stack trace) and non-panic 5xx
// responses to Sentry, tagged with request_id and the chi route pattern (never
// the raw path or any PII). It must sit INNER to chi's Recoverer: a panic is
// captured here, re-panicked (Repanic: true), and Recoverer turns it into the
// 500 response. A no-DSN Init makes the SDK a no-op, so this is safe in dev.
func SentryMiddleware(next http.Handler) http.Handler {
	handler := sentryhttp.New(sentryhttp.Options{Repanic: true})
	return handler.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
			hub.Scope().SetTag("request_id", middleware.GetReqID(r.Context()))
		}

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		// Non-panic 5xx: capture a message with the route pattern. (Panics never
		// reach this line — sentryhttp captures and re-panics them above.)
		if ww.Status() >= 500 {
			if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
				route := chi.RouteContext(r.Context()).RoutePattern()
				hub.Scope().SetTag("route", route)
				hub.CaptureMessage(fmt.Sprintf("http %d: %s", ww.Status(), route))
			}
		}
	}))
}
