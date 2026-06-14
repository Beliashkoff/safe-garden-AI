package admin

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/mailer"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/observability"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// Bootstrapped reports whether the operator account exists. The SPA uses it to
// choose between the sign-in screen and the first-time setup wizard.
func (s *Service) Bootstrapped(ctx context.Context) (bool, error) {
	n, err := s.store.CountAdmins(ctx)
	if err != nil {
		return false, fmt.Errorf("admin: count admins: %w", err)
	}
	return n > 0, nil
}

// RequestSetupCode emails a one-time setup code. Allowed only before the
// account exists and only for the pinned ADMIN_EMAIL address.
func (s *Service) RequestSetupCode(ctx context.Context, rawEmail string) error {
	ok, err := s.Bootstrapped(ctx)
	if err != nil {
		return err
	}
	if ok {
		return ErrAlreadyBootstrapped
	}
	return s.issueCode(ctx, rawEmail, mailer.AdminCodePurposeSetup)
}

// CompleteSetup verifies the setup code, creates the operator account with the
// chosen password and opens the first session.
func (s *Service) CompleteSetup(ctx context.Context, rawEmail, code, password string, dev DeviceMeta) (SessionResult, error) {
	ok, err := s.Bootstrapped(ctx)
	if err != nil {
		return SessionResult{}, err
	}
	if ok {
		return SessionResult{}, ErrAlreadyBootstrapped
	}
	email, err := s.allowedEmail(rawEmail)
	if err != nil {
		return SessionResult{}, err
	}
	if !validPassword(password) {
		return SessionResult{}, ErrWeakPassword
	}
	if err := s.consumeCode(ctx, email, mailer.AdminCodePurposeSetup, code); err != nil {
		return SessionResult{}, err
	}

	hash, err := authpkg.HashPassword(password)
	if err != nil {
		return SessionResult{}, fmt.Errorf("admin: hash password: %w", err)
	}
	adminUser, err := s.store.CreateAdmin(ctx, db.CreateAdminParams{Email: email, PasswordHash: hash})
	if err != nil {
		return SessionResult{}, fmt.Errorf("admin: create admin: %w", err)
	}
	s.audit(ctx, adminUser.ID, "admin_setup", "", "", dev.IP)
	return s.openSession(ctx, adminUser, dev)
}

// Login signs the operator in with email+password. Failures are uniform
// (ErrInvalidCredentials) regardless of which check failed, audited, counted in
// the brute-force metric, and rate-limited. Two guards back the throttle: a
// per-IP counter and an account-global counter — the latter holds even when an
// attacker rotates spoofed X-Forwarded-For values to defeat the per-IP one.
func (s *Service) Login(ctx context.Context, rawEmail, password string, dev DeviceMeta) (SessionResult, error) {
	if limited, err := s.loginRateLimited(ctx, dev); err != nil {
		return SessionResult{}, err
	} else if limited {
		return SessionResult{}, ErrRateLimited
	}

	email := normalizeEmail(rawEmail)
	adminUser, lookupErr := s.store.GetAdminByEmail(ctx, email)
	if lookupErr != nil && !errors.Is(lookupErr, pgx.ErrNoRows) {
		return SessionResult{}, fmt.Errorf("admin: get admin: %w", lookupErr)
	}

	// Constant-time authorization: always run a bcrypt comparison (against the
	// real hash on a hit, the dummy hash on a miss) and a constant-time email
	// compare, so response latency does not reveal whether ADMIN_EMAIL matched.
	hash := s.dummyHash
	if lookupErr == nil {
		hash = adminUser.PasswordHash
	}
	pwOK := authpkg.VerifyPassword(password, hash) == nil
	emailOK := subtle.ConstantTimeCompare([]byte(email), []byte(s.adminEmail)) == 1
	authorized := lookupErr == nil && emailOK && pwOK

	if !authorized {
		observability.IncAdminLoginFailure()
		s.audit(ctx, uuid.Nil, "admin_login_failed", "", "", dev.IP)
		return SessionResult{}, ErrInvalidCredentials
	}

	if err := s.store.TouchAdminLogin(ctx, adminUser.ID); err != nil {
		s.logger.WarnContext(ctx, "touch admin login failed", "err", err.Error())
	}
	s.audit(ctx, adminUser.ID, "admin_login", "", "", dev.IP)
	return s.openSession(ctx, adminUser, dev)
}

// loginRateLimited reports whether the brute-force window is exhausted, by IP
// (when a valid one is available) or globally for the single operator account.
func (s *Service) loginRateLimited(ctx context.Context, dev DeviceMeta) (bool, error) {
	since := timestamptz(s.now().Add(-failedLoginWindow))
	if dev.IP.IsValid() {
		ipp := dev.IP
		n, err := s.store.CountRecentFailedAdminLogins(ctx, db.CountRecentFailedAdminLoginsParams{
			Ip:        &ipp,
			CreatedAt: since,
		})
		if err != nil {
			return false, fmt.Errorf("admin: failed-login count (ip): %w", err)
		}
		if n >= maxFailedLogins {
			return true, nil
		}
	}
	gn, err := s.store.CountRecentFailedAdminLoginsGlobal(ctx, since)
	if err != nil {
		return false, fmt.Errorf("admin: failed-login count (global): %w", err)
	}
	return gn >= maxFailedLoginsGlobal, nil
}

// Logout revokes the presented session. Idempotent: an unknown or already
// revoked token is not an error.
func (s *Service) Logout(ctx context.Context, token string) error {
	sess, err := s.store.GetAdminSessionByHash(ctx, authpkg.HashRefreshToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("admin: get session: %w", err)
	}
	if err := s.store.RevokeAdminSession(ctx, sess.ID); err != nil {
		return fmt.Errorf("admin: revoke session: %w", err)
	}
	return nil
}

// RequestResetCode emails a password-reset code — only to ADMIN_EMAIL and only
// once the account exists.
func (s *Service) RequestResetCode(ctx context.Context, rawEmail string) error {
	ok, err := s.Bootstrapped(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotBootstrapped
	}
	if err := s.issueCode(ctx, rawEmail, mailer.AdminCodePurposeReset); err != nil {
		return err
	}
	s.audit(ctx, uuid.Nil, "admin_reset_requested", "", "", netip.Addr{})
	return nil
}

// CompleteReset verifies the reset code, sets the new password and revokes
// every session. The operator signs in again with the new password.
func (s *Service) CompleteReset(ctx context.Context, rawEmail, code, newPassword string) error {
	email, err := s.allowedEmail(rawEmail)
	if err != nil {
		return err
	}
	adminUser, err := s.store.GetAdminByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotBootstrapped
	}
	if err != nil {
		return fmt.Errorf("admin: get admin: %w", err)
	}
	if !validPassword(newPassword) {
		return ErrWeakPassword
	}
	if err := s.consumeCode(ctx, email, mailer.AdminCodePurposeReset, code); err != nil {
		return err
	}
	hash, err := authpkg.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("admin: hash password: %w", err)
	}
	if err := s.store.UpdateAdminPassword(ctx, db.UpdateAdminPasswordParams{ID: adminUser.ID, PasswordHash: hash}); err != nil {
		return fmt.Errorf("admin: update password: %w", err)
	}
	if err := s.store.RevokeAllAdminSessions(ctx, adminUser.ID); err != nil {
		return fmt.Errorf("admin: revoke sessions: %w", err)
	}
	s.audit(ctx, adminUser.ID, "admin_password_reset", "", "", netip.Addr{})
	return nil
}

// ChangePassword verifies the current password, sets the new one and revokes
// every OTHER session (the current one stays signed in).
func (s *Service) ChangePassword(ctx context.Context, adminID, sessionID uuid.UUID, current, newPassword string, dev DeviceMeta) error {
	adminUser, err := s.store.GetAdminByID(ctx, adminID)
	if err != nil {
		return fmt.Errorf("admin: get admin: %w", err)
	}
	if authpkg.VerifyPassword(current, adminUser.PasswordHash) != nil {
		return ErrInvalidCredentials
	}
	if !validPassword(newPassword) {
		return ErrWeakPassword
	}
	hash, err := authpkg.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("admin: hash password: %w", err)
	}
	if err := s.store.UpdateAdminPassword(ctx, db.UpdateAdminPasswordParams{ID: adminUser.ID, PasswordHash: hash}); err != nil {
		return fmt.Errorf("admin: update password: %w", err)
	}
	if err := s.store.RevokeOtherAdminSessions(ctx, db.RevokeOtherAdminSessionsParams{AdminID: adminUser.ID, ID: sessionID}); err != nil {
		return fmt.Errorf("admin: revoke other sessions: %w", err)
	}
	s.audit(ctx, adminUser.ID, "admin_password_changed", "", "", dev.IP)
	return nil
}

// ValidateSession resolves a raw cookie token to its session, sliding the
// expiry forward when the session has been quiet for a while.
func (s *Service) ValidateSession(ctx context.Context, token string) (db.AdminSession, error) {
	if token == "" {
		return db.AdminSession{}, ErrSessionInvalid
	}
	sess, err := s.store.GetAdminSessionByHash(ctx, authpkg.HashRefreshToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return db.AdminSession{}, ErrSessionInvalid
	}
	if err != nil {
		return db.AdminSession{}, fmt.Errorf("admin: get session: %w", err)
	}
	// Absolute lifetime cap: a continuously-used session cannot live past
	// absoluteSessionMax regardless of the sliding expiry, forcing periodic
	// re-authentication even if a token leaks.
	if s.now().Sub(sess.CreatedAt.Time) > absoluteSessionMax {
		if err := s.store.RevokeAdminSession(ctx, sess.ID); err != nil {
			s.logger.WarnContext(ctx, "revoke over-age session failed", "err", err.Error())
		}
		return db.AdminSession{}, ErrSessionInvalid
	}
	if s.now().Sub(sess.LastSeenAt.Time) > sessionTouchInterval {
		if err := s.store.TouchAdminSession(ctx, db.TouchAdminSessionParams{
			ID:        sess.ID,
			ExpiresAt: timestamptz(s.now().Add(s.sessionTTL)),
		}); err != nil {
			s.logger.WarnContext(ctx, "touch admin session failed", "err", err.Error())
		}
	}
	return sess, nil
}

// Me returns the operator account view.
func (s *Service) Me(ctx context.Context, adminID uuid.UUID) (AdminView, error) {
	adminUser, err := s.store.GetAdminByID(ctx, adminID)
	if err != nil {
		return AdminView{}, fmt.Errorf("admin: get admin: %w", err)
	}
	return toAdminView(adminUser), nil
}

// ListSessions returns the active sessions, flagging the current one.
func (s *Service) ListSessions(ctx context.Context, adminID, currentSessionID uuid.UUID) ([]SessionView, error) {
	rows, err := s.store.ListActiveAdminSessions(ctx, adminID)
	if err != nil {
		return nil, fmt.Errorf("admin: list sessions: %w", err)
	}
	out := make([]SessionView, 0, len(rows))
	for _, r := range rows {
		v := SessionView{
			ID:         r.ID,
			UserAgent:  r.UserAgent.String,
			CreatedAt:  r.CreatedAt.Time,
			LastSeenAt: r.LastSeenAt.Time,
			Current:    r.ID == currentSessionID,
		}
		if r.Ip != nil {
			v.IP = r.Ip.String()
		}
		out = append(out, v)
	}
	return out, nil
}

// RevokeSession revokes one of the operator's own sessions.
func (s *Service) RevokeSession(ctx context.Context, adminID, sessionID uuid.UUID, dev DeviceMeta) error {
	rows, err := s.store.ListActiveAdminSessions(ctx, adminID)
	if err != nil {
		return fmt.Errorf("admin: list sessions: %w", err)
	}
	for _, r := range rows {
		if r.ID == sessionID {
			if err := s.store.RevokeAdminSession(ctx, sessionID); err != nil {
				return fmt.Errorf("admin: revoke session: %w", err)
			}
			s.audit(ctx, adminID, "admin_session_revoked", "session", sessionID.String(), dev.IP)
			return nil
		}
	}
	return ErrSessionInvalid
}

// --- helpers ---

// allowedEmail normalizes and pins the address to ADMIN_EMAIL.
func (s *Service) allowedEmail(rawEmail string) (string, error) {
	email := normalizeEmail(rawEmail)
	if email == "" || email != s.adminEmail {
		return "", ErrEmailNotAllowed
	}
	return email, nil
}

// issueCode rate-limits, generates, persists and emails a one-time code.
func (s *Service) issueCode(ctx context.Context, rawEmail, purpose string) error {
	email, err := s.allowedEmail(rawEmail)
	if err != nil {
		return err
	}
	n, err := s.store.CountRecentAdminCodes(ctx, db.CountRecentAdminCodesParams{
		Email:     email,
		Purpose:   purpose,
		CreatedAt: timestamptz(s.now().Add(-time.Hour)),
	})
	if err != nil {
		return fmt.Errorf("admin: count codes: %w", err)
	}
	if n >= maxCodesPerHour {
		return ErrRateLimited
	}
	code, hash, err := authpkg.GenerateOTP()
	if err != nil {
		return fmt.Errorf("admin: generate code: %w", err)
	}
	if _, err := s.store.CreateAdminCode(ctx, db.CreateAdminCodeParams{
		Email:     email,
		Purpose:   purpose,
		CodeHash:  hash,
		ExpiresAt: timestamptz(s.now().Add(codeTTL)),
	}); err != nil {
		return fmt.Errorf("admin: persist code: %w", err)
	}
	if err := s.mailer.SendAdminCode(ctx, email, code, purpose); err != nil {
		return fmt.Errorf("admin: send code: %w", err)
	}
	return nil
}

// consumeCode verifies and burns the active code for (email, purpose).
func (s *Service) consumeCode(ctx context.Context, email, purpose, code string) error {
	row, err := s.store.GetActiveAdminCode(ctx, db.GetActiveAdminCodeParams{Email: email, Purpose: purpose})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidCode
	}
	if err != nil {
		return fmt.Errorf("admin: get code: %w", err)
	}
	attempts, err := s.store.IncrementAdminCodeAttempts(ctx, row.ID)
	if err != nil {
		return fmt.Errorf("admin: bump attempts: %w", err)
	}
	if attempts > maxCodeAttempts {
		return ErrTooManyAttempts
	}
	if err := authpkg.VerifyOTP(code, row.CodeHash); err != nil {
		return ErrInvalidCode
	}
	if err := s.store.MarkAdminCodeUsed(ctx, row.ID); err != nil {
		return fmt.Errorf("admin: mark code used: %w", err)
	}
	return nil
}

// openSession mints an opaque token (same construction as refresh tokens:
// 256-bit random, sha256 stored) and persists the session row.
func (s *Service) openSession(ctx context.Context, adminUser db.AdminUser, dev DeviceMeta) (SessionResult, error) {
	raw, hash, err := authpkg.NewRefreshToken()
	if err != nil {
		return SessionResult{}, fmt.Errorf("admin: new session token: %w", err)
	}
	expires := s.now().Add(s.sessionTTL)
	params := db.CreateAdminSessionParams{
		AdminID:   adminUser.ID,
		TokenHash: hash,
		UserAgent: textOrNull(dev.UserAgent),
		ExpiresAt: timestamptz(expires),
	}
	if dev.IP.IsValid() {
		ip := dev.IP
		params.Ip = &ip
	}
	if _, err := s.store.CreateAdminSession(ctx, params); err != nil {
		return SessionResult{}, fmt.Errorf("admin: persist session: %w", err)
	}
	return SessionResult{Token: raw, ExpiresAt: expires, Admin: toAdminView(adminUser)}, nil
}
