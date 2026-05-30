package cleanup

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

type fakeStore struct {
	pending     []uuid.UUID
	marked      []uuid.UUID
	unused      []db.Upload
	deletedRows []string
	audits      []string
	markErr     error
	deleteErr   error
}

func (f *fakeStore) ListUsersPendingMediaPurge(_ context.Context, limit int32) ([]uuid.UUID, error) {
	if int(limit) < len(f.pending) {
		return f.pending[:limit], nil
	}
	return f.pending, nil
}

func (f *fakeStore) MarkUserMediaPurged(_ context.Context, id uuid.UUID) error {
	if f.markErr != nil {
		return f.markErr
	}
	f.marked = append(f.marked, id)
	return nil
}

func (f *fakeStore) ListUnusedUploadsBefore(_ context.Context, _ pgtype.Timestamptz) ([]db.Upload, error) {
	return f.unused, nil
}

func (f *fakeStore) DeleteUpload(_ context.Context, key string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletedRows = append(f.deletedRows, key)
	return nil
}

func (f *fakeStore) InsertAuditLog(_ context.Context, arg db.InsertAuditLogParams) error {
	f.audits = append(f.audits, arg.Action)
	return nil
}

type fakeObjs struct {
	prefixes    []string
	deletedKeys []string
	prefixErr   error
	keysErr     error
}

func (f *fakeObjs) DeletePrefix(_ context.Context, prefix string) (int, error) {
	if f.prefixErr != nil {
		return 0, f.prefixErr
	}
	f.prefixes = append(f.prefixes, prefix)
	return 3, nil
}

func (f *fakeObjs) DeleteKeys(_ context.Context, keys []string) error {
	if f.keysErr != nil {
		return f.keysErr
	}
	f.deletedKeys = append(f.deletedKeys, keys...)
	return nil
}

func newSvc(s store, o objDeleter) *Service {
	svc := NewService(s, o, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.now = func() time.Time { return time.Unix(1_800_000_000, 0).UTC() }
	return svc
}

func TestPurgeDeletedUserMedia_HappyPath(t *testing.T) {
	id := uuid.New()
	fs := &fakeStore{pending: []uuid.UUID{id}}
	fo := &fakeObjs{}

	n, err := newSvc(fs, fo).PurgeDeletedUserMedia(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"u/" + id.String() + "/"}, fo.prefixes)
	assert.Equal(t, []uuid.UUID{id}, fs.marked)
	assert.Contains(t, fs.audits, "account_media_purged")
}

func TestPurgeDeletedUserMedia_ObjErrLeavesUnmarked(t *testing.T) {
	id := uuid.New()
	fs := &fakeStore{pending: []uuid.UUID{id}}
	fo := &fakeObjs{prefixErr: errors.New("s3 down")}

	n, err := newSvc(fs, fo).PurgeDeletedUserMedia(context.Background())
	require.NoError(t, err) // object errors are non-fatal
	assert.Equal(t, 0, n)
	assert.Empty(t, fs.marked, "user stays pending for retry")
	assert.Empty(t, fs.audits)
}

func TestGCUnusedUploads_DeletesObjectThenRow(t *testing.T) {
	fs := &fakeStore{unused: []db.Upload{
		{StorageKey: "u/a/img/1.jpg"},
		{StorageKey: "u/b/img/2.jpg"},
	}}
	fo := &fakeObjs{}

	n, err := newSvc(fs, fo).GCUnusedUploads(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.ElementsMatch(t, []string{"u/a/img/1.jpg", "u/b/img/2.jpg"}, fo.deletedKeys)
	assert.ElementsMatch(t, []string{"u/a/img/1.jpg", "u/b/img/2.jpg"}, fs.deletedRows)
}

func TestGCUnusedUploads_ObjErrSkipsRow(t *testing.T) {
	fs := &fakeStore{unused: []db.Upload{{StorageKey: "u/a/img/1.jpg"}}}
	fo := &fakeObjs{keysErr: errors.New("s3 down")}

	n, err := newSvc(fs, fo).GCUnusedUploads(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, fs.deletedRows, "row kept when object delete fails")
}

func TestRunOnce_RunsBoth(t *testing.T) {
	id := uuid.New()
	fs := &fakeStore{
		pending: []uuid.UUID{id},
		unused:  []db.Upload{{StorageKey: "u/a/img/1.jpg"}},
	}
	fo := &fakeObjs{}

	require.NoError(t, newSvc(fs, fo).RunOnce(context.Background()))
	assert.Len(t, fs.marked, 1)
	assert.Len(t, fs.deletedRows, 1)
}
