// Package cleanup runs the asynchronous media-deletion and upload-GC jobs
// (ROADMAP §3.2). It is invoked once per run by cmd/cleanup (scheduled by cron);
// it does not own a scheduler. All work is idempotent so a crashed or repeated
// run is safe: deleted-but-unpurged users are retried until media_purged_at is
// set, and deleting a missing object is a no-op.
package cleanup

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

const (
	defaultRetention = 7 * 24 * time.Hour // unused-upload TTL (ROADMAP §3.2)
	defaultBatch     = 100                // users scanned per purge run
)

// store is the subset of storage.Store the cleanup job needs (consumer-side).
type store interface {
	ListUsersPendingMediaPurge(ctx context.Context, limit int32) ([]uuid.UUID, error)
	MarkUserMediaPurged(ctx context.Context, id uuid.UUID) error
	ListUnusedUploadsBefore(ctx context.Context, createdAt pgtype.Timestamptz) ([]db.Upload, error)
	DeleteUpload(ctx context.Context, storageKey string) error
	InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) error
}

// objDeleter removes objects from object storage (satisfied by *objstore.Client).
type objDeleter interface {
	DeletePrefix(ctx context.Context, prefix string) (deleted int, err error)
	DeleteKeys(ctx context.Context, keys []string) error
}

// Service runs the two cleanup jobs.
type Service struct {
	store     store
	objs      objDeleter
	logger    *slog.Logger
	now       func() time.Time
	retention time.Duration
	batch     int32
}

func NewService(s store, objs objDeleter, logger *slog.Logger) *Service {
	return &Service{
		store:     s,
		objs:      objs,
		logger:    logger,
		now:       time.Now,
		retention: defaultRetention,
		batch:     defaultBatch,
	}
}

// RunOnce executes both jobs and logs a summary. A fatal (DB) error is returned;
// per-item object-storage failures are logged and retried on the next run.
func (s *Service) RunOnce(ctx context.Context) error {
	purged, err := s.PurgeDeletedUserMedia(ctx)
	if err != nil {
		return fmt.Errorf("cleanup: purge deleted media: %w", err)
	}
	gc, err := s.GCUnusedUploads(ctx)
	if err != nil {
		return fmt.Errorf("cleanup: gc unused uploads: %w", err)
	}
	s.logger.InfoContext(ctx, "cleanup run complete", "users_purged", purged, "uploads_gc", gc)
	return nil
}

// PurgeDeletedUserMedia deletes the Object Storage prefix u/{user_id}/ for each
// deleted-but-unpurged user, then marks it purged. An object-storage failure
// leaves the user unmarked so the next run retries it.
func (s *Service) PurgeDeletedUserMedia(ctx context.Context) (int, error) {
	ids, err := s.store.ListUsersPendingMediaPurge(ctx, s.batch)
	if err != nil {
		return 0, fmt.Errorf("list pending purge: %w", err)
	}
	purged := 0
	for _, id := range ids {
		prefix := "u/" + id.String() + "/"
		n, err := s.objs.DeletePrefix(ctx, prefix)
		if err != nil {
			s.logger.ErrorContext(ctx, "purge prefix failed; will retry next run",
				"user_id", id.String(), "err", err.Error())
			continue
		}
		if err := s.store.MarkUserMediaPurged(ctx, id); err != nil {
			return purged, fmt.Errorf("mark purged: %w", err)
		}
		s.audit(ctx, id, "account_media_purged")
		s.logger.InfoContext(ctx, "purged user media", "user_id", id.String(), "objects", n)
		purged++
	}
	return purged, nil
}

// GCUnusedUploads deletes presigned-but-never-attached uploads older than the
// retention window (objects first, then rows). A failed object delete leaves the
// row for the next run.
func (s *Service) GCUnusedUploads(ctx context.Context) (int, error) {
	cutoff := pgtype.Timestamptz{Time: s.now().Add(-s.retention), Valid: true}
	rows, err := s.store.ListUnusedUploadsBefore(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("list unused uploads: %w", err)
	}
	deleted := 0
	for _, up := range rows {
		if err := s.objs.DeleteKeys(ctx, []string{up.StorageKey}); err != nil {
			s.logger.ErrorContext(ctx, "gc object delete failed; will retry next run",
				"err", err.Error())
			continue
		}
		if err := s.store.DeleteUpload(ctx, up.StorageKey); err != nil {
			return deleted, fmt.Errorf("delete upload row: %w", err)
		}
		deleted++
	}
	return deleted, nil
}

// audit records a system-level deletion event (no IP, no details — CLAUDE.md #3).
func (s *Service) audit(ctx context.Context, userID uuid.UUID, action string) {
	if err := s.store.InsertAuditLog(ctx, db.InsertAuditLogParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Action: action,
	}); err != nil {
		s.logger.ErrorContext(ctx, "audit insert failed", "action", action, "err", err.Error())
	}
}
