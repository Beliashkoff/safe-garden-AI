package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// GetAccount returns the authenticated user's profile.
func (s *Service) GetAccount(ctx context.Context, userID uuid.UUID) (UserView, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserView{}, ErrUserNotFound
	}
	if err != nil {
		return UserView{}, fmt.Errorf("auth: get account: %w", err)
	}
	return toView(user), nil
}

// DeleteAccount soft-deletes the user (nulling unique identifiers so they can be
// reused), revokes all refresh tokens, and writes an audit row — atomically.
// Object Storage cleanup of the user's media prefix is deferred to stage 3.
func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID, dev DeviceMeta) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
			return fmt.Errorf("auth: revoke tokens: %w", err)
		}
		if err := q.SoftDeleteUser(ctx, userID); err != nil {
			return fmt.Errorf("auth: soft delete: %w", err)
		}
		s.audit(ctx, q, userID, "account_deleted", dev.IP)
		return nil
	})
}
