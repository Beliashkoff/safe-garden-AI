package ratelimit

import (
	"context"
	"log/slog"

	"github.com/go-redis/redis_rate/v10"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// messagesPerSecond is the per-user cap on POST /v1/messages (ARCH §8.2).
const messagesPerSecond = 20

// RedisLimiter enforces per-user request rates via redis_rate (GCRA token
// bucket) on a shared Redis, so the limit holds across multiple API instances.
type RedisLimiter struct {
	limiter *redis_rate.Limiter
	logger  *slog.Logger
}

// NewRedis builds a RedisLimiter over an existing go-redis client.
func NewRedis(rdb *redis.Client, logger *slog.Logger) *RedisLimiter {
	return &RedisLimiter{limiter: redis_rate.NewLimiter(rdb), logger: logger}
}

// AllowMessage reports whether userID may send another message now.
//
// Fail-open: if Redis is unavailable we log and allow the request — a rate
// limiter outage must not take chat down. The DB and worker still bound abuse.
func (l *RedisLimiter) AllowMessage(ctx context.Context, userID uuid.UUID) (bool, error) {
	res, err := l.limiter.Allow(ctx, "msg:"+userID.String(), redis_rate.PerSecond(messagesPerSecond))
	if err != nil {
		l.logger.ErrorContext(ctx, "rate limiter unavailable, allowing request", "err", err.Error())
		return true, nil
	}
	return res.Allowed > 0, nil
}
