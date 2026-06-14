package admin

import (
	"context"
	"fmt"
	"sort"
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
	return o, nil
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
