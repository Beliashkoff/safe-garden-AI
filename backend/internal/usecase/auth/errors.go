// Package auth is the authentication usecase: it orchestrates the auth
// primitives (JWT, Yandex ID / VK ID OAuth, OTP, refresh tokens) and storage
// into the flows behind the HTTP endpoints — sign-in (Yandex/VK/email),
// refresh rotation, logout, and account read/delete. The transport layer maps
// these sentinel errors to HTTP responses; this package has no transport
// dependency.
package auth

import "errors"

var (
	// ErrInvalidOAuthState — the state is unknown, expired, already consumed,
	// or belongs to another provider. One error for all cases by design.
	ErrInvalidOAuthState = errors.New("auth: invalid or expired oauth state")
	// ErrOAuthFailed — the provider rejected the code/token (tampered, expired,
	// or issued to another application).
	ErrOAuthFailed = errors.New("auth: oauth sign-in failed")
	// ErrOAuthUnavailable — the provider could not be reached (network/5xx);
	// the client should retry later.
	ErrOAuthUnavailable = errors.New("auth: oauth provider unavailable")
	// ErrInvalidEmail — email is syntactically invalid or too long.
	ErrInvalidEmail = errors.New("auth: invalid email")
	// ErrInvalidOTP — code is wrong, expired, used, or never issued. The single
	// error avoids leaking which case occurred.
	ErrInvalidOTP = errors.New("auth: invalid or expired code")
	// ErrTooManyAttempts — the ≤5 attempts-per-code cap was exceeded.
	ErrTooManyAttempts = errors.New("auth: too many attempts")
	// ErrRateLimited — OTP request quota exceeded (≤3/hour/email).
	ErrRateLimited = errors.New("auth: rate limited")
	// ErrInvalidToken — refresh token unknown, expired, or already used.
	ErrInvalidToken = errors.New("auth: invalid refresh token")
	// ErrUserNotFound — authenticated user_id has no live row (e.g. deleted).
	ErrUserNotFound = errors.New("auth: user not found")
)
