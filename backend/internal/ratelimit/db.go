package ratelimit

import (
	"context"
	"fmt"
)

// maxOTPRequestsPerHour is the ≤3-per-hour-per-email cap from ARCH §8.1.
const maxOTPRequestsPerHour = 3

// emailCodeCounter is the storage dependency the DB limiter needs. It is
// satisfied by the sqlc *db.Queries (CountRecentEmailCodes counts rows created
// in the last hour for an email).
type emailCodeCounter interface {
	CountRecentEmailCodes(ctx context.Context, email string) (int64, error)
}

type dbLimiter struct {
	counter emailCodeCounter
}

// NewDB returns a Limiter backed by the email_codes table. It is the baseline
// enforcement; the same cap will later be fronted by Redis for speed.
func NewDB(counter emailCodeCounter) Limiter {
	return &dbLimiter{counter: counter}
}

func (l *dbLimiter) AllowEmailOTPRequest(ctx context.Context, email string) (bool, error) {
	n, err := l.counter.CountRecentEmailCodes(ctx, email)
	if err != nil {
		return false, fmt.Errorf("ratelimit: count recent codes: %w", err)
	}
	return n < maxOTPRequestsPerHour, nil
}
