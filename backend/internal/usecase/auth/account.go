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

// DeleteAccount erases the user's content and anonymizes the account, all in one
// transaction (SPEC F9 / ARCH §6.3): revoke refresh tokens, hard-delete the
// conversation (cascades to messages + message_blocks) and upload rows, then
// soft-delete the user (nulling unique identifiers so the email/OAuth subject
// can be reused) and write an audit row. usage_log is kept for billing.
//
// The user's Object Storage objects (prefix u/{user_id}/) are removed
// asynchronously by the cleanup job — the soft-deleted row stays with
// media_purged_at = NULL until then (CLAUDE.md invariant #8, §8.3).
func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID, dev DeviceMeta) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
			return fmt.Errorf("auth: revoke tokens: %w", err)
		}
		if err := q.DeleteUserConversations(ctx, userID); err != nil {
			return fmt.Errorf("auth: delete conversations: %w", err)
		}
		if err := q.DeleteUserUploads(ctx, userID); err != nil {
			return fmt.Errorf("auth: delete uploads: %w", err)
		}
		if err := q.SoftDeleteUser(ctx, userID); err != nil {
			return fmt.Errorf("auth: soft delete: %w", err)
		}
		s.audit(ctx, q, userID, "account_deleted", dev.IP)
		return nil
	})
}
