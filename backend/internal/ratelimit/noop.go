package ratelimit

import "context"

// noop allows everything. Used in tests and any context where a Limiter is
// required structurally but no quota applies yet.
type noop struct{}

// NewNoop returns a Limiter that never denies.
func NewNoop() Limiter { return noop{} }

func (noop) AllowEmailOTPRequest(context.Context, string) (bool, error) { return true, nil }
