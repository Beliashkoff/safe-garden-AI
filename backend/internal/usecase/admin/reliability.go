package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// This file backs the "Надёжность" admin page: LLM-worker health, dependency
// health, failed-turn breakdown and the threshold alert banner. Worker and
// dependency probes are injected (consumer-side interfaces) and wired in
// cmd/api; results are cached briefly so the dashboard and the alert evaluator
// do not hammer the worker/services.

const healthCacheTTL = 15 * time.Second

// WorkerPinger probes the llm-worker for liveness. Implemented by
// *llm.WorkerClient (GET /healthz over the same mTLS transport as chat).
type WorkerPinger interface {
	Ping(ctx context.Context) error
}

// DependencyChecker probes the backing services (PG/Redis/S3/SpeechKit/SMTP).
// Implemented by the adapter in cmd/api, which holds the real clients.
type DependencyChecker interface {
	Check(ctx context.Context) []DepStatus
}

// AlertThresholds gates the banner alerts. A zero threshold disables that rule.
type AlertThresholds struct {
	ErrorsPerHour int
	FailRatePct   int
	DailyCostUSD  float64
}

// Ops bundles the optional reliability dependencies, set after construction.
type Ops struct {
	WorkerPinger WorkerPinger
	WorkerModel  string
	Deps         DependencyChecker
	Alerts       AlertThresholds
}

// SetOps wires the reliability dependencies. Called once from cmd/api.
func (s *Service) SetOps(o Ops) {
	s.workerPinger = o.WorkerPinger
	s.workerModel = o.WorkerModel
	s.deps = o.Deps
	s.alerts = o.Alerts
}

// DepStatus is one backing-service health light.
type DepStatus struct {
	Name       string
	Configured bool
	Up         bool
	LatencyMS  int64
	Detail     string
}

// WorkerHealth is the llm-worker traffic light plus the requested model.
type WorkerHealth struct {
	Configured bool
	Online     bool
	LatencyMS  int64
	Model      string
	CheckedAt  time.Time
}

// GetWorkerHealth pings the worker (cached ~15s). When no pinger is wired (dev /
// mock client) it reports Configured=false but still surfaces the model.
func (s *Service) GetWorkerHealth(ctx context.Context) WorkerHealth {
	if s.workerPinger == nil {
		return WorkerHealth{Model: s.workerModel}
	}
	s.healthMu.Lock()
	if s.workerCache != nil && s.now().Sub(s.workerCacheAt) < healthCacheTTL {
		c := *s.workerCache
		s.healthMu.Unlock()
		return c
	}
	s.healthMu.Unlock()

	start := s.now()
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	err := s.workerPinger.Ping(pingCtx)
	cancel()
	h := WorkerHealth{
		Configured: true,
		Online:     err == nil,
		LatencyMS:  s.now().Sub(start).Milliseconds(),
		Model:      s.workerModel,
		CheckedAt:  s.now(),
	}
	s.healthMu.Lock()
	s.workerCache = &h
	s.workerCacheAt = s.now()
	s.healthMu.Unlock()
	return h
}

// GetDependencies returns the backing-service health lights (cached ~15s).
func (s *Service) GetDependencies(ctx context.Context) []DepStatus {
	if s.deps == nil {
		return nil
	}
	s.healthMu.Lock()
	if s.depCache != nil && s.now().Sub(s.depCacheAt) < healthCacheTTL {
		c := s.depCache
		s.healthMu.Unlock()
		return c
	}
	s.healthMu.Unlock()

	res := s.deps.Check(ctx)
	s.healthMu.Lock()
	s.depCache = res
	s.depCacheAt = s.now()
	s.healthMu.Unlock()
	return res
}

// FailCodeStat is one failed-turn reason bucket.
type FailCodeStat struct {
	Code  string
	Count int64
}

// GetFailCodes returns the failed assistant-turn breakdown for the last `days`.
func (s *Service) GetFailCodes(ctx context.Context, days int) ([]FailCodeStat, error) {
	rows, err := s.store.FailCodesSince(ctx, s.sinceDays(days, 14))
	if err != nil {
		return nil, fmt.Errorf("admin: fail codes: %w", err)
	}
	out := make([]FailCodeStat, 0, len(rows))
	for _, r := range rows {
		out = append(out, FailCodeStat{Code: r.FailCode, Count: r.Count})
	}
	return out, nil
}

// Alert is one active threshold breach for the admin banner.
type Alert struct {
	Level   string // warning | error
	Metric  string
	Message string
}

// minFailRateSample avoids crying wolf on a tiny denominator (e.g. 1 of 2).
const minFailRateSample = 20

// EvaluateAlerts checks the configured thresholds against current aggregates and
// the worker light. Returns the active alerts (empty when all clear / disabled).
func (s *Service) EvaluateAlerts(ctx context.Context) ([]Alert, error) {
	now := s.now()
	var alerts []Alert

	if s.alerts.ErrorsPerHour > 0 {
		n, err := s.store.CountErrorEventsSince(ctx, timestamptz(now.Add(-time.Hour)))
		if err != nil {
			return nil, fmt.Errorf("admin: alert errors: %w", err)
		}
		if n > int64(s.alerts.ErrorsPerHour) {
			alerts = append(alerts, Alert{
				Level:   "error",
				Metric:  "errors_1h",
				Message: fmt.Sprintf("Серверных ошибок за час: %d (порог %d)", n, s.alerts.ErrorsPerHour),
			})
		}
	}

	if s.alerts.FailRatePct > 0 {
		ans, err := s.store.MessageStatusCountsSince(ctx, timestamptz(now.AddDate(0, 0, -1)))
		if err != nil {
			return nil, fmt.Errorf("admin: alert fail rate: %w", err)
		}
		total := ans.Complete + ans.Failed + ans.Cancelled
		if total >= minFailRateSample {
			rate := float64(ans.Failed) / float64(total) * 100
			if rate > float64(s.alerts.FailRatePct) {
				alerts = append(alerts, Alert{
					Level:   "error",
					Metric:  "fail_rate_24h",
					Message: fmt.Sprintf("Доля сбоев ответов за сутки: %.0f%% (порог %d%%)", rate, s.alerts.FailRatePct),
				})
			}
		}
	}

	if s.alerts.DailyCostUSD > 0 {
		nowUTC := now.UTC()
		dayStart := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
		c, err := s.store.SumCostBetween(ctx, db.SumCostBetweenParams{
			FromTs: timestamptz(dayStart),
			ToTs:   timestamptz(nowUTC),
		})
		if err != nil {
			return nil, fmt.Errorf("admin: alert daily cost: %w", err)
		}
		if cost := numericToFloat(c); cost > s.alerts.DailyCostUSD {
			alerts = append(alerts, Alert{
				Level:   "warning",
				Metric:  "daily_cost",
				Message: fmt.Sprintf("Расход на ИИ за сегодня: $%.2f (порог $%.2f)", cost, s.alerts.DailyCostUSD),
			})
		}
	}

	if wh := s.GetWorkerHealth(ctx); wh.Configured && !wh.Online {
		alerts = append(alerts, Alert{
			Level:   "error",
			Metric:  "worker_offline",
			Message: "LLM-worker (Frankfurt) недоступен",
		})
	}

	return alerts, nil
}
