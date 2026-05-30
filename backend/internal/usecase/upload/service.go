package upload

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

const (
	maxImageBytes = 10 * 1024 * 1024 // ARCH §8.2: image ≤ 10 MB
	presignTTL    = 5 * time.Minute  // ARCH §5: presigned PUT TTL
)

// imageExt is the allowed image content-type whitelist (ARCH §8.2) → extension.
var imageExt = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/heic": "heic",
}

// presigner issues presigned PUT/GET URLs (consumer-side interface; satisfied
// by *objstore.Client).
type presigner interface {
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (url string, headers map[string]string, err error)
	PresignGet(ctx context.Context, key string, ttl time.Duration) (url string, err error)
}

// uploadStore records pending uploads (consumer-side interface; satisfied by
// *storage.Store).
type uploadStore interface {
	CreateUpload(ctx context.Context, arg db.CreateUploadParams) (db.Upload, error)
}

// Service issues presigned upload URLs and records the pending upload.
type Service struct {
	store  uploadStore
	objs   presigner
	ttl    time.Duration
	now    func() time.Time
	newKey func() string
}

func NewService(store uploadStore, objs presigner) *Service {
	return &Service{
		store:  store,
		objs:   objs,
		ttl:    presignTTL,
		now:    time.Now,
		newKey: uuid.NewString,
	}
}

// Presign validates the request, records the pending upload (used=false), then
// presigns a PUT URL. The storage key is owner-scoped (`u/{user_id}/img/...`),
// which both enables prefix-based deletion on account removal and lets the chat
// usecase verify ownership later.
func (s *Service) Presign(ctx context.Context, userID uuid.UUID, in PresignInput) (PresignOutput, error) {
	ext, ok := imageExt[in.ContentType]
	if !ok {
		return PresignOutput{}, ErrUnsupportedType
	}
	switch {
	case in.SizeBytes <= 0:
		return PresignOutput{}, ErrInvalidSize
	case in.SizeBytes > maxImageBytes:
		return PresignOutput{}, ErrTooLarge
	}

	key := fmt.Sprintf("u/%s/img/%s.%s", userID.String(), s.newKey(), ext)

	// Record before presigning: a row with no object is harmless (GC removes it
	// via ListUnusedUploadsBefore), but a URL with no row would be untrackable.
	if _, err := s.store.CreateUpload(ctx, db.CreateUploadParams{
		UserID:      userID,
		StorageKey:  key,
		ContentType: in.ContentType,
		SizeBytes:   in.SizeBytes,
	}); err != nil {
		return PresignOutput{}, fmt.Errorf("upload: record: %w", err)
	}

	url, headers, err := s.objs.PresignPut(ctx, key, in.ContentType, s.ttl)
	if err != nil {
		return PresignOutput{}, fmt.Errorf("upload: presign: %w", err)
	}
	return PresignOutput{
		URL:       url,
		Key:       key,
		Headers:   headers,
		ExpiresAt: s.now().Add(s.ttl),
	}, nil
}

// PresignView issues a short-lived presigned GET URL for an owned object, so the
// client can display a stored photo it no longer has cached locally. Ownership
// is enforced by the owner-scoped key prefix (`u/{user_id}/`), the same scheme
// the chat usecase uses to verify image_ref blocks; a key outside the caller's
// prefix yields ErrNotOwner (mapped to 403) without revealing whether it exists.
func (s *Service) PresignView(ctx context.Context, userID uuid.UUID, storageKey string) (ViewOutput, error) {
	if storageKey == "" || !strings.HasPrefix(storageKey, "u/"+userID.String()+"/") {
		return ViewOutput{}, ErrNotOwner
	}
	url, err := s.objs.PresignGet(ctx, storageKey, s.ttl)
	if err != nil {
		return ViewOutput{}, fmt.Errorf("upload: presign view: %w", err)
	}
	return ViewOutput{URL: url, ExpiresAt: s.now().Add(s.ttl)}, nil
}
