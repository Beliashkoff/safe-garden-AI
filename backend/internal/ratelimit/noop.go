package ratelimit

import (
	"context"

	"github.com/google/uuid"
)

// noop allows everything. Used in tests and any context where a Limiter is
// required structurally but no quota applies yet.
type noop struct{}

// NewNoop returns a Limiter that never denies.
func NewNoop() Limiter { return noop{} }

func (noop) AllowEmailOTPRequest(context.Context, string) (bool, error) { return true, nil }

// noopMessage allows every message. Used in dev when REDIS_ADDR is unset.
type noopMessage struct{}

// NewNoopMessage returns a message limiter that never denies.
func NewNoopMessage() *noopMessage { return &noopMessage{} }

func (*noopMessage) AllowMessage(context.Context, uuid.UUID) (bool, error) { return true, nil }
