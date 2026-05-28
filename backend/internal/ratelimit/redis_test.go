package ratelimit

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func TestRedisLimiter_AllowsUpToCapThenDenies(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	limiter := NewRedis(newTestRedis(t), logger)
	user := uuid.New()
	ctx := context.Background()

	allowed := 0
	// GCRA admits a burst equal to the rate; fire 25 and count allows.
	for i := 0; i < 25; i++ {
		ok, err := limiter.AllowMessage(ctx, user)
		require.NoError(t, err)
		if ok {
			allowed++
		}
	}
	assert.Equal(t, messagesPerSecond, allowed, "burst is capped at the per-second rate")

	// A different user is independent.
	ok, err := limiter.AllowMessage(ctx, uuid.New())
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRedisLimiter_FailOpenWhenRedisDown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rdb := newTestRedis(t)
	require.NoError(t, rdb.Close()) // force connection errors

	ok, err := NewRedis(rdb, logger).AllowMessage(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.True(t, ok, "rate limiter outage must fail open")
}
