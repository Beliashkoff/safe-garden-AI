// Package ctxkey holds typed context keys shared across the HTTP transport so
// values stored by middleware (e.g. the authenticated user_id) are read back
// type-safely instead of via stringly-typed lookups.
package ctxkey

import (
	"context"

	"github.com/google/uuid"
)

type contextKey int

const userIDKey contextKey = iota

// WithUserID returns a child context carrying the authenticated user_id. Set by
// the RequireAuth middleware after a successful JWT verification.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserID extracts the authenticated user_id. ok is false when no authenticated
// user is present — handlers behind RequireAuth can rely on ok being true.
func UserID(ctx context.Context) (id uuid.UUID, ok bool) {
	id, ok = ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
