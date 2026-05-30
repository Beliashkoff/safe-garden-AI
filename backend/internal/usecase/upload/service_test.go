package upload

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

type fakeStore struct {
	created []db.CreateUploadParams
	err     error
}

func (f *fakeStore) CreateUpload(_ context.Context, arg db.CreateUploadParams) (db.Upload, error) {
	f.created = append(f.created, arg)
	if f.err != nil {
		return db.Upload{}, f.err
	}
	return db.Upload{StorageKey: arg.StorageKey, UserID: arg.UserID, ContentType: arg.ContentType, SizeBytes: arg.SizeBytes}, nil
}

type fakePresigner struct {
	gotKey    string
	gotGetKey string
}

func (f *fakePresigner) PresignPut(_ context.Context, key, contentType string, _ time.Duration) (string, map[string]string, error) {
	f.gotKey = key
	return "http://put/" + key, map[string]string{"Content-Type": contentType}, nil
}

func (f *fakePresigner) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	f.gotGetKey = key
	return "http://get/" + key, nil
}

func newSvc(store uploadStore, p presigner) *Service {
	s := NewService(store, p)
	s.newKey = func() string { return "fixed" }
	return s
}

func TestPresign_Validation(t *testing.T) {
	uid := uuid.New()
	ctx := context.Background()

	_, err := newSvc(&fakeStore{}, &fakePresigner{}).Presign(ctx, uid, PresignInput{ContentType: "application/pdf", SizeBytes: 10})
	assert.ErrorIs(t, err, ErrUnsupportedType)

	_, err = newSvc(&fakeStore{}, &fakePresigner{}).Presign(ctx, uid, PresignInput{ContentType: "image/jpeg", SizeBytes: 0})
	assert.ErrorIs(t, err, ErrInvalidSize)

	_, err = newSvc(&fakeStore{}, &fakePresigner{}).Presign(ctx, uid, PresignInput{ContentType: "image/jpeg", SizeBytes: maxImageBytes + 1})
	assert.ErrorIs(t, err, ErrTooLarge)
}

func TestPresign_HappyPath(t *testing.T) {
	uid := uuid.New()
	store := &fakeStore{}
	out, err := newSvc(store, &fakePresigner{}).Presign(context.Background(), uid, PresignInput{ContentType: "image/jpeg", SizeBytes: 1000})
	require.NoError(t, err)

	wantKey := "u/" + uid.String() + "/img/fixed.jpg"
	assert.Equal(t, wantKey, out.Key)
	assert.Equal(t, "http://put/"+wantKey, out.URL)
	assert.Equal(t, "image/jpeg", out.Headers["Content-Type"])
	require.Len(t, store.created, 1)
	assert.Equal(t, wantKey, store.created[0].StorageKey)
	assert.Equal(t, int64(1000), store.created[0].SizeBytes)
	assert.False(t, out.ExpiresAt.IsZero())
}

func TestPresign_RecordFailure(t *testing.T) {
	store := &fakeStore{err: errors.New("db down")}
	_, err := newSvc(store, &fakePresigner{}).Presign(context.Background(), uuid.New(), PresignInput{ContentType: "image/png", SizeBytes: 10})
	require.Error(t, err)
}

func TestPresignView_HappyPath(t *testing.T) {
	uid := uuid.New()
	p := &fakePresigner{}
	key := "u/" + uid.String() + "/img/abc.jpg"

	out, err := newSvc(&fakeStore{}, p).PresignView(context.Background(), uid, key)
	require.NoError(t, err)
	assert.Equal(t, "http://get/"+key, out.URL)
	assert.Equal(t, key, p.gotGetKey)
	assert.False(t, out.ExpiresAt.IsZero())
}

func TestPresignView_RejectsForeignKey(t *testing.T) {
	uid := uuid.New()
	other := uuid.New()
	p := &fakePresigner{}

	_, err := newSvc(&fakeStore{}, p).PresignView(context.Background(), uid, "u/"+other.String()+"/img/abc.jpg")
	assert.ErrorIs(t, err, ErrNotOwner)

	_, err = newSvc(&fakeStore{}, p).PresignView(context.Background(), uid, "")
	assert.ErrorIs(t, err, ErrNotOwner)

	assert.Empty(t, p.gotGetKey, "must not presign a key the user does not own")
}
