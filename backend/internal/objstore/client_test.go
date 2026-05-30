package objstore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_RequiresBucket(t *testing.T) {
	_, err := New(Config{})
	require.Error(t, err)
}

// PresignPut is local SigV4 signing (no network), so it is fully unit-testable.
func TestPresignPut(t *testing.T) {
	c, err := New(Config{
		Endpoint:     "http://localhost:9000",
		Region:       "ru-central1",
		AccessKey:    "key",
		SecretKey:    "secret",
		Bucket:       "media",
		UsePathStyle: true,
	})
	require.NoError(t, err)

	url, headers, err := c.PresignPut(context.Background(), "u/123/img/a.jpg", "image/jpeg", 5*time.Minute)
	require.NoError(t, err)
	assert.Contains(t, url, "u/123/img/a.jpg")
	assert.Contains(t, url, "media") // bucket, path-style
	assert.Contains(t, url, "X-Amz-Signature")
	assert.Equal(t, "image/jpeg", headers["Content-Type"])
}

func TestDisabled(t *testing.T) {
	d := Disabled{}
	_, _, err := d.PresignPut(context.Background(), "k", "image/jpeg", time.Minute)
	assert.ErrorIs(t, err, ErrDisabled)
	_, _, err = d.Get(context.Background(), "k")
	assert.ErrorIs(t, err, ErrDisabled)
}
