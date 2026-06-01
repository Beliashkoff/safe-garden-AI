package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scrapeMetrics(t *testing.T) string {
	t.Helper()
	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.String()
}

// TestMetricsMiddleware_UsesRoutePattern is the cardinality-safety guard: the
// route label must be the chi pattern (/messages/{id}), never the concrete id —
// otherwise one series leaks per UUID.
func TestMetricsMiddleware_UsesRoutePattern(t *testing.T) {
	r := chi.NewRouter()
	r.Use(Metrics)
	r.Get("/messages/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/messages/abc-123-xyz", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	body := scrapeMetrics(t)
	assert.Contains(t, body, `route="/messages/{id}"`, "must label by route pattern")
	assert.NotContains(t, body, "abc-123-xyz", "raw path id must never become a label")
}

func TestMetricsMiddleware_UnmatchedRouteCollapses(t *testing.T) {
	r := chi.NewRouter()
	r.Use(Metrics)
	r.Get("/exists", func(w http.ResponseWriter, _ *http.Request) {})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/no-such-route", nil))
	require.Equal(t, http.StatusNotFound, rec.Code)

	assert.Contains(t, scrapeMetrics(t), `route="unmatched"`)
}

func TestObserveClaudeTurn_RecordsSeries(t *testing.T) {
	ObserveClaudeTurn(100, 50, 0.01, 2*time.Second, "")
	ObserveClaudeTurn(0, 0, 0, time.Second, "upstream_error")

	body := scrapeMetrics(t)
	assert.Contains(t, body, "claude_requests_total")
	assert.Contains(t, body, `claude_tokens_total{direction="in"}`)
	assert.Contains(t, body, `claude_tokens_total{direction="out"}`)
	assert.Contains(t, body, `claude_request_errors_total{code="upstream_error"}`)
	assert.Contains(t, body, "claude_cost_usd_total")
}

func TestIncHelpers_RecordSeries(t *testing.T) {
	IncMessage("complete")
	IncUpload("presigned")

	body := scrapeMetrics(t)
	assert.Contains(t, body, `message_total{status="complete"}`)
	assert.Contains(t, body, `upload_status_total{status="presigned"}`)
}
