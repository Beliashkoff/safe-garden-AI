package admin

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// Overview is the dashboard headline numbers.
type Overview struct {
	UsersTotal    int64
	UsersNew7d    int64
	MessagesTotal int64
	Messages7d    int64
	TokensIn30d   int64
	TokensOut30d  int64
	CostUSD30d    float64
	Taps30d       int64
	CatalogTotal  int64
	CatalogActive int64
	Errors24h     int64

	// Active users (distinct senders) over trailing windows.
	DAU int64
	WAU int64
	MAU int64

	// Answer quality: thumbs up/down over 30d and coverage (rated / completed).
	FeedbackUp30d       int64
	FeedbackDown30d     int64
	FeedbackCoverage30d float64

	// Assistant-turn reliability over 7d (success rate = complete / terminal).
	AnswersComplete7d  int64
	AnswersFailed7d    int64
	AnswersCancelled7d int64

	// Account-deletion -> media-purge pipeline (compliance / store review).
	AccountsDeletedTotal  int64
	MediaPurgePending     int64
	MediaPurgeOldestHours float64

	// Cost: month-to-date, linear forecast for the month, previous month.
	CostMTD           float64
	CostForecastMonth float64
	CostPrevMonth     float64
}

// DayPoint is one bucket of a per-day series.
type DayPoint struct {
	Day       time.Time
	Users     int64
	Messages  int64
	TokensIn  int64
	TokensOut int64
	CostUSD   float64
}

// TapStat is taps per product for the dashboard top list.
type TapStat struct {
	Slug string
	Taps int64
}

// ErrorEventView is one row of the "Ошибки" feed.
type ErrorEventView struct {
	ID        int64
	Source    string
	Route     string
	Method    string
	Status    int32
	RequestID string
	Message   string
	CreatedAt time.Time
}

// AuditView is one row of the "Журнал действий" feed.
type AuditView struct {
	ID        int64
	AdminID   string
	Action    string
	Entity    string
	EntityID  string
	IP        string
	CreatedAt time.Time
}

// FeedbackPoint is one day of the thumbs up/down series.
type FeedbackPoint struct {
	Day  time.Time
	Up   int64
	Down int64
}

// MessageStatusPoint is one day of assistant-turn terminal statuses.
type MessageStatusPoint struct {
	Day       time.Time
	Complete  int64
	Failed    int64
	Cancelled int64
}

// ProviderStat is sign-in volume for one RU login method.
type ProviderStat struct {
	Provider string // yandex | vk | email
	Logins   int64
	Users    int64
}

// CostUserStat is one of the most expensive users. User is a masked id prefix.
type CostUserStat struct {
	User      string
	Requests  int64
	TokensIn  int64
	TokensOut int64
	CostUSD   float64
}

// ErrorRouteStat is a (route, status) bucket of the server error feed.
type ErrorRouteStat struct {
	Route  string
	Status int32
	Count  int64
}

// ErrorDayPoint is one day of the server error count series.
type ErrorDayPoint struct {
	Day   time.Time
	Count int64
}

// ErrorBreakdown groups the error feed by route/status and by day.
type ErrorBreakdown struct {
	ByRoute []ErrorRouteStat
	ByDay   []ErrorDayPoint
}

// GetOverview aggregates the dashboard headline numbers.
func (s *Service) GetOverview(ctx context.Context) (Overview, error) {
	now := s.now()
	var o Overview
	var err error

	if o.UsersTotal, err = s.store.CountActiveUsers(ctx); err != nil {
		return o, fmt.Errorf("admin: count users: %w", err)
	}
	if o.UsersNew7d, err = s.store.CountUsersCreatedSince(ctx, timestamptz(now.AddDate(0, 0, -7))); err != nil {
		return o, fmt.Errorf("admin: count new users: %w", err)
	}
	if o.MessagesTotal, err = s.store.CountUserMessages(ctx); err != nil {
		return o, fmt.Errorf("admin: count messages: %w", err)
	}
	if o.Messages7d, err = s.store.CountUserMessagesSince(ctx, timestamptz(now.AddDate(0, 0, -7))); err != nil {
		return o, fmt.Errorf("admin: count recent messages: %w", err)
	}
	usage, err := s.store.SumUsageSince(ctx, timestamptz(now.AddDate(0, 0, -30)))
	if err != nil {
		return o, fmt.Errorf("admin: sum usage: %w", err)
	}
	o.TokensIn30d = usage.TokensIn
	o.TokensOut30d = usage.TokensOut
	o.CostUSD30d = numericToFloat(usage.CostUsd)
	if o.Taps30d, err = s.store.CountFertilizerTapsSince(ctx, timestamptz(now.AddDate(0, 0, -30))); err != nil {
		return o, fmt.Errorf("admin: count taps: %w", err)
	}
	catalog, err := s.store.CountFertilizers(ctx)
	if err != nil {
		return o, fmt.Errorf("admin: count catalog: %w", err)
	}
	o.CatalogTotal = catalog.Total
	o.CatalogActive = catalog.Active
	if o.Errors24h, err = s.store.CountErrorEventsSince(ctx, timestamptz(now.Add(-24*time.Hour))); err != nil {
		return o, fmt.Errorf("admin: count errors: %w", err)
	}

	if err := s.fillEngagement(ctx, &o, now); err != nil {
		return o, err
	}
	if err := s.fillCostForecast(ctx, &o); err != nil {
		return o, err
	}
	return o, nil
}

// fillEngagement populates active-user windows, feedback, answer success rate and
// the deletion pipeline. Split out of GetOverview to keep its complexity in check.
func (s *Service) fillEngagement(ctx context.Context, o *Overview, now time.Time) error {
	active, err := s.store.ActiveUserWindows(ctx, db.ActiveUserWindowsParams{
		DaySince:   timestamptz(now.AddDate(0, 0, -1)),
		WeekSince:  timestamptz(now.AddDate(0, 0, -7)),
		MonthSince: timestamptz(now.AddDate(0, 0, -30)),
	})
	if err != nil {
		return fmt.Errorf("admin: active user windows: %w", err)
	}
	o.DAU, o.WAU, o.MAU = active.Dau, active.Wau, active.Mau

	fb, err := s.store.FeedbackTotalsSince(ctx, timestamptz(now.AddDate(0, 0, -30)))
	if err != nil {
		return fmt.Errorf("admin: feedback totals: %w", err)
	}
	o.FeedbackUp30d, o.FeedbackDown30d = fb.Up, fb.Down

	answers30d, err := s.store.MessageStatusCountsSince(ctx, timestamptz(now.AddDate(0, 0, -30)))
	if err != nil {
		return fmt.Errorf("admin: answer counts 30d: %w", err)
	}
	if answers30d.Complete > 0 {
		o.FeedbackCoverage30d = float64(fb.Up+fb.Down) / float64(answers30d.Complete)
	}

	answers7d, err := s.store.MessageStatusCountsSince(ctx, timestamptz(now.AddDate(0, 0, -7)))
	if err != nil {
		return fmt.Errorf("admin: answer counts 7d: %w", err)
	}
	o.AnswersComplete7d = answers7d.Complete
	o.AnswersFailed7d = answers7d.Failed
	o.AnswersCancelled7d = answers7d.Cancelled

	pipeline, err := s.store.GetDeletionPipeline(ctx)
	if err != nil {
		return fmt.Errorf("admin: deletion pipeline: %w", err)
	}
	o.AccountsDeletedTotal = pipeline.DeletedTotal
	o.MediaPurgePending = pipeline.PurgePending
	o.MediaPurgeOldestHours = pipeline.OldestPendingHours
	return nil
}

// fillCostForecast computes month-to-date Claude spend, a linear projection for
// the full month, and the previous month's spend. Month boundaries are in UTC to
// match the date_trunc('day', ..., 'UTC') bucketing used by the daily series.
func (s *Service) fillCostForecast(ctx context.Context, o *Overview) error {
	now := s.now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonth := monthStart.AddDate(0, 1, 0)
	prevMonth := monthStart.AddDate(0, -1, 0)

	mtd, err := s.store.SumCostBetween(ctx, db.SumCostBetweenParams{
		FromTs: timestamptz(monthStart),
		ToTs:   timestamptz(now),
	})
	if err != nil {
		return fmt.Errorf("admin: cost mtd: %w", err)
	}
	prev, err := s.store.SumCostBetween(ctx, db.SumCostBetweenParams{
		FromTs: timestamptz(prevMonth),
		ToTs:   timestamptz(monthStart),
	})
	if err != nil {
		return fmt.Errorf("admin: cost prev month: %w", err)
	}
	o.CostMTD = numericToFloat(mtd)
	o.CostPrevMonth = numericToFloat(prev)

	elapsed := now.Sub(monthStart).Hours()
	full := nextMonth.Sub(monthStart).Hours()
	if elapsed > 0 {
		o.CostForecastMonth = o.CostMTD * full / elapsed
	}
	return nil
}

// GetTimeseries returns merged per-day series for the last `days` days.
func (s *Service) GetTimeseries(ctx context.Context, days int) ([]DayPoint, error) {
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365
	}
	since := timestamptz(s.now().AddDate(0, 0, -days).Truncate(24 * time.Hour))

	users, err := s.store.UsersByDay(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("admin: users by day: %w", err)
	}
	messages, err := s.store.MessagesByDay(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("admin: messages by day: %w", err)
	}
	usage, err := s.store.UsageByDay(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("admin: usage by day: %w", err)
	}

	merged := map[time.Time]*DayPoint{}
	point := func(day time.Time) *DayPoint {
		key := day.UTC()
		if p, ok := merged[key]; ok {
			return p
		}
		p := &DayPoint{Day: key}
		merged[key] = p
		return p
	}
	for _, r := range users {
		point(r.Day.Time).Users = r.Count
	}
	for _, r := range messages {
		point(r.Day.Time).Messages = r.Count
	}
	for _, r := range usage {
		p := point(r.Day.Time)
		p.TokensIn = r.TokensIn
		p.TokensOut = r.TokensOut
		p.CostUSD = numericToFloat(r.CostUsd)
	}

	out := make([]DayPoint, 0, len(merged))
	for _, p := range merged {
		out = append(out, *p)
	}
	sortDayPoints(out)
	return out, nil
}

// GetTopProducts returns the most tapped catalog cards for the last `days`.
func (s *Service) GetTopProducts(ctx context.Context, days int) ([]TapStat, error) {
	if days < 1 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	rows, err := s.store.TopFertilizerTaps(ctx, timestamptz(s.now().AddDate(0, 0, -days)))
	if err != nil {
		return nil, fmt.Errorf("admin: top taps: %w", err)
	}
	out := make([]TapStat, 0, len(rows))
	for _, r := range rows {
		out = append(out, TapStat{Slug: r.Slug, Taps: r.Taps})
	}
	return out, nil
}

// GetFeedbackSeries returns the daily thumbs up/down counts for the last `days`.
func (s *Service) GetFeedbackSeries(ctx context.Context, days int) ([]FeedbackPoint, error) {
	rows, err := s.store.FeedbackByDay(ctx, s.sinceDays(days, 30))
	if err != nil {
		return nil, fmt.Errorf("admin: feedback by day: %w", err)
	}
	out := make([]FeedbackPoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, FeedbackPoint{Day: r.Day.Time, Up: r.Up, Down: r.Down})
	}
	return out, nil
}

// GetMessageStatusSeries returns daily assistant-turn terminal statuses for `days`.
func (s *Service) GetMessageStatusSeries(ctx context.Context, days int) ([]MessageStatusPoint, error) {
	rows, err := s.store.MessageStatusByDay(ctx, s.sinceDays(days, 30))
	if err != nil {
		return nil, fmt.Errorf("admin: message status by day: %w", err)
	}
	out := make([]MessageStatusPoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, MessageStatusPoint{Day: r.Day.Time, Complete: r.Complete, Failed: r.Failed, Cancelled: r.Cancelled})
	}
	return out, nil
}

// GetLoginBreakdown returns sign-in volume per RU provider for the last `days`.
func (s *Service) GetLoginBreakdown(ctx context.Context, days int) ([]ProviderStat, error) {
	rows, err := s.store.LoginsByProviderSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return nil, fmt.Errorf("admin: logins by provider: %w", err)
	}
	out := make([]ProviderStat, 0, len(rows))
	for _, r := range rows {
		out = append(out, ProviderStat{
			Provider: strings.TrimPrefix(r.Action, "sign_in_"),
			Logins:   r.Logins,
			Users:    r.Users,
		})
	}
	return out, nil
}

// GetTopCostUsers returns the most expensive users by Claude spend for `days`. The
// real UUID never leaves the backend: only an 8-char hex prefix is returned, enough
// to correlate rows without exposing the identity (CLAUDE.md invariant #3/#10).
func (s *Service) GetTopCostUsers(ctx context.Context, days int) ([]CostUserStat, error) {
	rows, err := s.store.TopCostUsersSince(ctx, s.sinceDays(days, 7))
	if err != nil {
		return nil, fmt.Errorf("admin: top cost users: %w", err)
	}
	out := make([]CostUserStat, 0, len(rows))
	for _, r := range rows {
		out = append(out, CostUserStat{
			User:      fmt.Sprintf("%x", r.UserID[:4]),
			Requests:  r.Requests,
			TokensIn:  r.TokensIn,
			TokensOut: r.TokensOut,
			CostUSD:   numericToFloat(r.CostUsd),
		})
	}
	return out, nil
}

// GetErrorBreakdown groups the server error feed by route/status and by day.
func (s *Service) GetErrorBreakdown(ctx context.Context, days int) (ErrorBreakdown, error) {
	since := s.sinceDays(days, 7)
	var b ErrorBreakdown
	routes, err := s.store.ErrorsByRouteSince(ctx, since)
	if err != nil {
		return b, fmt.Errorf("admin: errors by route: %w", err)
	}
	for _, r := range routes {
		b.ByRoute = append(b.ByRoute, ErrorRouteStat{Route: r.Route, Status: r.Status, Count: r.Count})
	}
	byDay, err := s.store.ErrorsByDaySince(ctx, since)
	if err != nil {
		return b, fmt.Errorf("admin: errors by day: %w", err)
	}
	for _, r := range byDay {
		b.ByDay = append(b.ByDay, ErrorDayPoint{Day: r.Day.Time, Count: r.Count})
	}
	return b, nil
}

// ListErrors returns the newest server error events.
func (s *Service) ListErrors(ctx context.Context, limit, offset int32) ([]ErrorEventView, error) {
	limit = clamp(limit, 1, 200)
	if offset < 0 {
		offset = 0
	}
	rows, err := s.store.ListErrorEvents(ctx, db.ListErrorEventsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("admin: list errors: %w", err)
	}
	out := make([]ErrorEventView, 0, len(rows))
	for _, r := range rows {
		out = append(out, ErrorEventView{
			ID:        r.ID,
			Source:    r.Source,
			Route:     r.Route,
			Method:    r.Method,
			Status:    r.Status,
			RequestID: r.RequestID.String,
			Message:   r.Message.String,
			CreatedAt: r.CreatedAt.Time,
		})
	}
	return out, nil
}

// ListAudit returns the newest admin actions.
func (s *Service) ListAudit(ctx context.Context, limit, offset int32) ([]AuditView, error) {
	limit = clamp(limit, 1, 200)
	if offset < 0 {
		offset = 0
	}
	rows, err := s.store.ListAdminAudit(ctx, db.ListAdminAuditParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("admin: list audit: %w", err)
	}
	out := make([]AuditView, 0, len(rows))
	for _, r := range rows {
		v := AuditView{
			ID:        r.ID,
			Action:    r.Action,
			Entity:    r.Entity.String,
			EntityID:  r.EntityID.String,
			CreatedAt: r.CreatedAt.Time,
		}
		if r.AdminID.Valid {
			v.AdminID = fmt.Sprintf("%x", r.AdminID.Bytes[:4]) // short prefix, enough for one operator
		}
		if r.Ip != nil {
			v.IP = r.Ip.String()
		}
		out = append(out, v)
	}
	return out, nil
}

// sinceDays clamps days to [1,365] (def when non-positive) and returns the UTC
// cutoff truncated to the day so daily buckets start cleanly.
func (s *Service) sinceDays(days, def int) pgtype.Timestamptz {
	if days < 1 {
		days = def
	}
	if days > 365 {
		days = 365
	}
	return timestamptz(s.now().AddDate(0, 0, -days).Truncate(24 * time.Hour))
}

func clamp(v, lo, hi int32) int32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func sortDayPoints(points []DayPoint) {
	sort.Slice(points, func(i, j int) bool { return points[i].Day.Before(points[j].Day) })
}

// numericToFloat converts a SQL NUMERIC aggregate to float64 for display.
// Display-only money (dashboard cost), so float precision is acceptable.
func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}
