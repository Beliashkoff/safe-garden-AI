// Package auth is the authentication usecase: it orchestrates the stage-1.1
// primitives (JWT, OIDC, OTP, refresh tokens) and storage into the flows behind
// the HTTP endpoints — sign-in (Apple/Google/email), refresh rotation, logout,
// and account read/delete. The transport layer maps these sentinel errors to
// HTTP responses; this package has no transport dependency.
package auth

import "errors"

var (
	// ErrInvalidIDToken — Apple/Google id_token failed verification.
	ErrInvalidIDToken = errors.New("auth: invalid id_token")
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
