// Package ratelimit gates abuse-prone actions. Stage 1.2 ships only the
// DB-baseline check required by ARCHITECTURE.md §8.1 (≤3 OTP requests per hour
// per email). The Limiter interface is the seam: stage 2.3 adds a Redis-backed
// implementation (per-IP login limits, faster OTP counting) behind the same
// interface with no handler changes.
package ratelimit

import "context"

// Limiter decides whether an action may proceed under its quota.
type Limiter interface {
	// AllowEmailOTPRequest reports whether another OTP may be issued for email
	// without breaching the per-hour cap. A non-nil error is a backend failure
	// (treated as fail-closed by callers), distinct from a quota denial (false).
	AllowEmailOTPRequest(ctx context.Context, email string) (bool, error)
}
