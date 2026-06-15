package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// Data-lifecycle & compliance views (152-FZ / 406-FZ) for the "Данные и
// комплаенс" page. Identifiers are masked before leaving the backend.

const (
	stalePurgeSLA   = 72 * time.Hour      // deleted-but-unpurged age that is "stuck"
	unusedUploadTTL = 7 * 24 * time.Hour  // GC window for orphaned uploads
	oldRevokedTTL   = 30 * 24 * time.Hour // revoked refresh tokens kept this long
	cleanupStaleTTL = 2 * time.Hour       // hourly cron; no run in 2h = a missed run
)

// DeletionEvent is one erasure-proof audit row (masked user).
type DeletionEvent struct {
	User      string
	Action    string
	CreatedAt time.Time
}

// ListDeletionEvents returns the newest account-deletion / media-purge events.
func (s *Service) ListDeletionEvents(ctx context.Context, limit, offset int32) ([]DeletionEvent, error) {
	limit = clamp(limit, 1, 500)
	if offset < 0 {
		offset = 0
	}
	rows, err := s.store.ListDeletionEvents(ctx, db.ListDeletionEventsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("admin: list deletion events: %w", err)
	}
	out := make([]DeletionEvent, 0, len(rows))
	for _, r := range rows {
		ev := DeletionEvent{Action: r.Action, CreatedAt: r.CreatedAt.Time}
		if r.UserID.Valid {
			ev.User = fmt.Sprintf("%x", r.UserID.Bytes[:4])
		}
		out = append(out, ev)
	}
	return out, nil
}

// StalePurge is a deleted account whose media purge is overdue (masked user).
type StalePurge struct {
	User         string
	DeletedAt    time.Time
	PendingHours float64
}

// ListStalePurges returns deleted accounts whose media purge is past the SLA.
func (s *Service) ListStalePurges(ctx context.Context) ([]StalePurge, error) {
	rows, err := s.store.ListStalePurges(ctx, timestamptz(s.now().Add(-stalePurgeSLA)))
	if err != nil {
		return nil, fmt.Errorf("admin: list stale purges: %w", err)
	}
	out := make([]StalePurge, 0, len(rows))
	for _, r := range rows {
		out = append(out, StalePurge{
			User:         fmt.Sprintf("%x", r.ID[:4]),
			DeletedAt:    r.DeletedAt.Time,
			PendingHours: r.PendingHours,
		})
	}
	return out, nil
}

// UploadGC is the orphaned-upload summary.
type UploadGC struct {
	UnusedTotal int64
	StaleTotal  int64
	StaleBytes  int64
}

// GetUploadGCStats returns orphaned-upload counts and the stale (overdue) subset.
func (s *Service) GetUploadGCStats(ctx context.Context) (UploadGC, error) {
	r, err := s.store.UploadGCStatsBefore(ctx, timestamptz(s.now().Add(-unusedUploadTTL)))
	if err != nil {
		return UploadGC{}, fmt.Errorf("admin: upload gc stats: %w", err)
	}
	return UploadGC{UnusedTotal: r.UnusedTotal, StaleTotal: r.StaleTotal, StaleBytes: r.StaleBytes}, nil
}

// RetentionBacklog counts service-table rows past their useful life.
type RetentionBacklog struct {
	ExpiredOTP   int64
	ExpiredOAuth int64
	OldRevoked   int64
}

// GetRetentionBacklog returns the data-minimisation backlog.
func (s *Service) GetRetentionBacklog(ctx context.Context) (RetentionBacklog, error) {
	r, err := s.store.RetentionBacklog(ctx, timestamptz(s.now().Add(-oldRevokedTTL)))
	if err != nil {
		return RetentionBacklog{}, fmt.Errorf("admin: retention backlog: %w", err)
	}
	return RetentionBacklog{ExpiredOTP: r.ExpiredOtp, ExpiredOAuth: r.ExpiredOauth, OldRevoked: r.OldRevoked}, nil
}

// UsageResidue is leftover usage_log tied to deleted users.
type UsageResidue struct {
	Rows        int64
	Users       int64
	OldestHours float64
	HasResidue  bool
}

// GetUsageResidue returns how much usage_log is still tied to deleted users.
func (s *Service) GetUsageResidue(ctx context.Context) (UsageResidue, error) {
	r, err := s.store.UsageResidueDeletedUsers(ctx)
	if err != nil {
		return UsageResidue{}, fmt.Errorf("admin: usage residue: %w", err)
	}
	res := UsageResidue{Rows: r.Rows, Users: r.Users, HasResidue: r.Rows > 0}
	if r.Oldest.Valid {
		res.OldestHours = s.now().Sub(r.Oldest.Time).Hours()
	}
	return res, nil
}

// CleanupHealth is the cleanup cron heartbeat.
type CleanupHealth struct {
	Recorded    bool
	LastRunAt   time.Time
	Stale       bool
	UsersPurged int64
	UploadsGC   int64
	AdminRowsGC int64
}

// GetCleanupHealth reports the last cleanup-cron run time + counts. Recorded is
// false until the cron has run at least once after the heartbeat shipped.
func (s *Service) GetCleanupHealth(ctx context.Context) (CleanupHealth, error) {
	row, err := s.store.GetLastCleanupRun(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CleanupHealth{}, nil
		}
		return CleanupHealth{}, fmt.Errorf("admin: cleanup health: %w", err)
	}
	h := CleanupHealth{Recorded: true, LastRunAt: row.CreatedAt.Time}
	h.Stale = s.now().Sub(h.LastRunAt) > cleanupStaleTTL
	var counts struct {
		UsersPurged int64 `json:"users_purged"`
		UploadsGC   int64 `json:"uploads_gc"`
		AdminRowsGC int64 `json:"admin_rows_gc"`
	}
	if len(row.Details) > 0 {
		_ = json.Unmarshal(row.Details, &counts)
	}
	h.UsersPurged = counts.UsersPurged
	h.UploadsGC = counts.UploadsGC
	h.AdminRowsGC = counts.AdminRowsGC
	return h, nil
}
