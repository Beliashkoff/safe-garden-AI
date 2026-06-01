package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry holds all application metrics. It is a dedicated registry (not the
// global default) so the exposed series are exactly the ones declared here — no
// accidental third-party series, and tests can reason about a known set.
var Registry = prometheus.NewRegistry()

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests by route pattern, method and status.",
		},
		[]string{"method", "route", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds by route pattern and method.",
			Buckets: []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "route"},
	)

	// Domain metrics (ARCH §9). Every label is a bounded enum — never user data,
	// ids, or message content (CLAUDE.md invariant #3).
	claudeRequestsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "claude_requests_total",
		Help: "Total Claude turns (denominator for the error-rate alert).",
	})
	claudeRequestErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "claude_request_errors_total",
			Help: "Failed Claude turns by failure code.",
		},
		[]string{"code"},
	)
	claudeRequestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "claude_request_duration_seconds",
		Help:    "End-to-end Claude turn latency as seen by the backend.",
		Buckets: []float64{.5, 1, 2, 5, 10, 20, 30, 60},
	})
	claudeTokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "claude_tokens_total",
			Help: "Claude tokens consumed by direction.",
		},
		[]string{"direction"}, // "in" | "out"
	)
	claudeCostUSDTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "claude_cost_usd_total",
		Help: "Estimated Claude spend in USD (derived from token counts).",
	})
	messageTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "message_total",
			Help: "Chat messages by terminal status.",
		},
		[]string{"status"}, // complete | failed | cancelled
	)
	uploadStatusTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "upload_status_total",
			Help: "Upload presign outcomes.",
		},
		[]string{"status"}, // presigned | rejected
	)
)

func init() {
	Registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		httpRequestsTotal,
		httpRequestDuration,
		claudeRequestsTotal,
		claudeRequestErrorsTotal,
		claudeRequestDuration,
		claudeTokensTotal,
		claudeCostUSDTotal,
		messageTotal,
		uploadStatusTotal,
	)
}

// Metrics is chi middleware recording http_requests_total and the duration
// histogram. The route label is the chi route PATTERN (e.g. /v1/messages/{id}),
// read AFTER next.ServeHTTP so chi has matched the leaf route — using the raw
// path would explode label cardinality with one series per id. Unmatched
// requests (404s) collapse to "unmatched".
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		httpRequestsTotal.WithLabelValues(r.Method, route, strconv.Itoa(ww.Status())).Inc()
		httpRequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}

// MetricsHandler serves the registry in the Prometheus text format. It is mounted
// on the dedicated internal metrics listener only — never exposed publicly.
func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{})
}

// ObserveClaudeTurn records one completed Claude turn. failCode is "" on success;
// a non-empty code (e.g. "upstream_error") also increments the error counter.
// Exported so usecase packages record metrics without importing prometheus.
func ObserveClaudeTurn(tokensIn, tokensOut int64, costUSD float64, dur time.Duration, failCode string) {
	claudeRequestsTotal.Inc()
	claudeRequestDuration.Observe(dur.Seconds())
	if tokensIn > 0 {
		claudeTokensTotal.WithLabelValues("in").Add(float64(tokensIn))
	}
	if tokensOut > 0 {
		claudeTokensTotal.WithLabelValues("out").Add(float64(tokensOut))
	}
	if costUSD > 0 {
		claudeCostUSDTotal.Add(costUSD)
	}
	if failCode != "" {
		claudeRequestErrorsTotal.WithLabelValues(failCode).Inc()
	}
}

// IncMessage records a chat message reaching a terminal status.
func IncMessage(status string) { messageTotal.WithLabelValues(status).Inc() }

// IncUpload records the outcome of an upload presign request.
func IncUpload(status string) { uploadStatusTotal.WithLabelValues(status).Inc() }
