package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// maxErrorMessageLen caps the stored error text. Error strings are
// operator-facing diagnostics (wrapped Go errors), never request bodies, but a
// runaway message must not bloat the table.
const maxErrorMessageLen = 500

// ErrorEvent is one server-side failure, as shown in the admin panel's
// "Ошибки" tab. Route is the chi route PATTERN (bounded cardinality, no IDs);
// Message carries the wrapped error chain — no PII by construction because
// error strings never embed message content/tokens (CLAUDE.md #3).
type ErrorEvent struct {
	Source    string
	Route     string
	Method    string
	Status    int
	RequestID string
	Message   string
}

// ErrorSink persists error events. Implemented by a thin adapter over the
// store in cmd/api; failures to record must never affect the response.
type ErrorSink interface {
	RecordError(ctx context.Context, ev ErrorEvent)
}

type errorNoteKey struct{}

type errorNote struct{ msg string }

// NoteError attaches the error text of the current request so the ErrorEvents
// middleware can persist it alongside the 5xx status. Called by httperr.Write;
// a no-op when the middleware is not installed (internal listeners, tests).
func NoteError(ctx context.Context, msg string) {
	if n, ok := ctx.Value(errorNoteKey{}).(*errorNote); ok {
		if len(msg) > maxErrorMessageLen {
			msg = msg[:maxErrorMessageLen]
		}
		n.msg = msg
	}
}

// ErrorEvents records every 5xx response into the sink (async, best-effort).
// Install it OUTSIDE Recoverer so panics-turned-500 are captured too.
func ErrorEvents(sink ErrorSink, source string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			note := &errorNote{}
			r = r.WithContext(context.WithValue(r.Context(), errorNoteKey{}, note))
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			if ww.Status() < 500 {
				return
			}
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			ev := ErrorEvent{
				Source:    source,
				Route:     route,
				Method:    r.Method,
				Status:    ww.Status(),
				RequestID: middleware.GetReqID(r.Context()),
				Message:   note.msg,
			}
			// Detached from the request lifecycle: recording must not delay the
			// response, and a cancelled request context must not lose the event.
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				sink.RecordError(ctx, ev)
			}()
		})
	}
}
