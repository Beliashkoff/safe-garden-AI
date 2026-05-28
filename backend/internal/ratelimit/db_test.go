package ratelimit

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCounter struct {
	n   int64
	err error
}

func (s stubCounter) CountRecentEmailCodes(context.Context, string) (int64, error) {
	return s.n, s.err
}

func TestDBLimiter_AllowsUnderCap(t *testing.T) {
	for _, n := range []int64{0, 1, 2} {
		ok, err := NewDB(stubCounter{n: n}).AllowEmailOTPRequest(context.Background(), "a@b.c")
		require.NoError(t, err)
		assert.True(t, ok, "n=%d should be allowed", n)
	}
}

func TestDBLimiter_DeniesAtCap(t *testing.T) {
	ok, err := NewDB(stubCounter{n: 3}).AllowEmailOTPRequest(context.Background(), "a@b.c")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestDBLimiter_FailsClosedOnError(t *testing.T) {
	ok, err := NewDB(stubCounter{err: errors.New("db down")}).AllowEmailOTPRequest(context.Background(), "a@b.c")
	require.Error(t, err)
	assert.False(t, ok)
}
