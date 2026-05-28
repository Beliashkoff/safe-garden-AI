package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// Refresh rotates a refresh token: the presented token is revoked and a new
// access+refresh pair is issued, atomically. Presenting an already-revoked or
// expired token is treated as theft — the entire token family for that user is
// revoked and the event is audited.
func (s *Service) Refresh(ctx context.Context, rawToken string, dev DeviceMeta) (AuthResult, error) {
	row, err := s.store.GetRefreshTokenByHash(ctx, authpkg.HashRefreshToken(rawToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthResult{}, ErrInvalidToken
	}
	if err != nil {
		return AuthResult{}, fmt.Errorf("auth: get refresh: %w", err)
	}

	if s.isReuse(row) {
		s.handleReuse(ctx, row, dev)
		return AuthResult{}, ErrInvalidToken
	}

	var result AuthResult
	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.RevokeRefreshToken(ctx, row.ID); err != nil {
			return fmt.Errorf("auth: revoke old refresh: %w", err)
		}
		user, err := q.GetUserByID(ctx, row.UserID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidToken
		}
		if err != nil {
			return fmt.Errorf("auth: get user: %w", err)
		}
		result, err = s.issueTokens(ctx, q, user, dev)
		return err
	})
	if err != nil {
		return AuthResult{}, err
	}
	return result, nil
}

// Logout revokes the presented refresh token. It is idempotent: unknown or
// already-revoked tokens succeed silently.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	row, err := s.store.GetRefreshTokenByHash(ctx, authpkg.HashRefreshToken(rawToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("auth: get refresh: %w", err)
	}
	if row.RevokedAt.Valid {
		return nil
	}
	if err := s.store.RevokeRefreshToken(ctx, row.ID); err != nil {
		return fmt.Errorf("auth: revoke refresh: %w", err)
	}
	return nil
}

func (s *Service) isReuse(row db.RefreshToken) bool {
	if row.RevokedAt.Valid {
		return true
	}
	return row.ExpiresAt.Valid && s.now().After(row.ExpiresAt.Time)
}

func (s *Service) handleReuse(ctx context.Context, row db.RefreshToken, dev DeviceMeta) {
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.RevokeAllUserRefreshTokens(ctx, row.UserID); err != nil {
			return err
		}
		s.audit(ctx, q, row.UserID, "refresh_reuse_detected", dev.IP)
		return nil
	})
	if err != nil {
		// Deliberately no user_id in the log line (ARCH §9 wants it hashed; the
		// audit_log row already records it). The DB error is enough to debug.
		s.logger.ErrorContext(ctx, "refresh reuse cleanup failed", "err", err.Error())
	}
}
