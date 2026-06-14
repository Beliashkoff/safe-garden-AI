package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// passwordBcryptCost — above DefaultCost (10) because admin passwords are
// long-lived, low-volume credentials guarding the whole catalog + stats
// surface: ~4x the work per guess costs us nothing at one login a day.
const passwordBcryptCost = 12

// ErrPasswordMismatch is returned by VerifyPassword on a wrong password.
// Callers must collapse it with "no such account" into one uniform error.
var ErrPasswordMismatch = errors.New("auth: password mismatch")

// HashPassword bcrypt-hashes an admin password. bcrypt silently uses only the
// first 72 bytes, so the caller validates length before getting here.
func HashPassword(password string) ([]byte, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), passwordBcryptCost)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}
	return h, nil
}

// VerifyPassword compares a plaintext password against the stored bcrypt hash.
func VerifyPassword(password string, hash []byte) error {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		return ErrPasswordMismatch
	}
	return nil
}
