package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// RequestOTP issues and emails a one-time code for email. It enforces the
// per-hour request cap before creating a code. The same flow serves sign-up and
// sign-in, so it reveals nothing about whether the address is already
// registered. A failure to send is surfaced (the client should retry).
func (s *Service) RequestOTP(ctx context.Context, rawEmail, locale string) error {
	email := normalizeEmail(rawEmail)
	if !validEmail(email) {
		return ErrInvalidEmail
	}

	allowed, err := s.limiter.AllowEmailOTPRequest(ctx, email)
	if err != nil {
		return fmt.Errorf("auth: rate check: %w", err)
	}
	if !allowed {
		return ErrRateLimited
	}

	code, hash, err := authpkg.GenerateOTP()
	if err != nil {
		return fmt.Errorf("auth: generate otp: %w", err)
	}
	if _, err := s.store.CreateEmailCode(ctx, db.CreateEmailCodeParams{
		Email:     email,
		CodeHash:  hash,
		ExpiresAt: timestamptz(s.now().Add(otpTTL)),
	}); err != nil {
		return fmt.Errorf("auth: persist otp: %w", err)
	}

	if err := s.mailer.SendOTP(ctx, email, code, locale); err != nil {
		return fmt.Errorf("auth: send otp: %w", err)
	}
	return nil
}

// VerifyOTP checks a submitted code and, on success, signs the user in
// (creating the account on first use). The attempt counter is incremented
// before the code comparison so the ≤5 cap holds even against wrong guesses.
func (s *Service) VerifyOTP(ctx context.Context, rawEmail, code string, dev DeviceMeta) (AuthResult, error) {
	email := normalizeEmail(rawEmail)
	if !validEmail(email) {
		return AuthResult{}, ErrInvalidEmail
	}
	if !validOTPFormat(code) {
		return AuthResult{}, ErrInvalidOTP
	}

	row, err := s.store.GetActiveEmailCode(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthResult{}, ErrInvalidOTP
	}
	if err != nil {
		return AuthResult{}, fmt.Errorf("auth: get code: %w", err)
	}

	attempts, err := s.store.IncrementEmailCodeAttempts(ctx, row.ID)
	if err != nil {
		return AuthResult{}, fmt.Errorf("auth: bump attempts: %w", err)
	}
	if attempts > maxOTPVerifyAttempts {
		return AuthResult{}, ErrTooManyAttempts
	}
	if err := authpkg.VerifyOTP(code, row.CodeHash); err != nil {
		return AuthResult{}, ErrInvalidOTP
	}

	var result AuthResult
	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.MarkEmailCodeUsed(ctx, row.ID); err != nil {
			return fmt.Errorf("auth: mark code used: %w", err)
		}
		user, err := s.resolveEmailUser(ctx, q, email)
		if err != nil {
			return err
		}
		result, err = s.issueTokens(ctx, q, user, dev)
		if err != nil {
			return err
		}
		s.audit(ctx, q, user.ID, "sign_in_email", dev.IP)
		return nil
	})
	if err != nil {
		return AuthResult{}, err
	}
	return result, nil
}

// resolveEmailUser returns the account for a verified email, creating it on
// first sign-in and upgrading email_verified when needed.
func (s *Service) resolveEmailUser(ctx context.Context, q *db.Queries, email string) (db.User, error) {
	user, err := q.GetUserByEmail(ctx, textOrNull(email))
	switch {
	case err == nil:
		if !user.EmailVerified {
			if err := q.MarkEmailVerified(ctx, user.ID); err != nil {
				return db.User{}, fmt.Errorf("auth: mark verified: %w", err)
			}
			user.EmailVerified = true
		}
		return user, nil
	case errors.Is(err, pgx.ErrNoRows):
		user, err := q.CreateUser(ctx, db.CreateUserParams{
			Email:         textOrNull(email),
			EmailVerified: true,
			DisplayName:   textOrNull(""),
			Column6:       "",
		})
		if err != nil {
			return db.User{}, fmt.Errorf("auth: create user: %w", err)
		}
		return user, nil
	default:
		return db.User{}, fmt.Errorf("auth: get user by email: %w", err)
	}
}
