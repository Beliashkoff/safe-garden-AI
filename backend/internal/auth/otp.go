package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

const (
	// otpDigits — 6 цифр согласно ARCHITECTURE §8.1.
	otpDigits = 6
	// otpRange — верхняя граница для rand.Int, 10^otpDigits.
	otpRange = 1_000_000
)

// ErrOTPMismatch is returned by VerifyOTP when the code does not match. We
// intentionally do not differentiate "wrong code" from "no code" at this
// layer — handlers should surface a single user-facing error to avoid
// leaking timing/cardinality signals.
var ErrOTPMismatch = errors.New("auth: otp mismatch")

// GenerateOTP returns a freshly sampled 6-digit code and its bcrypt hash.
//
// rand.Int gives a uniform distribution over [0, 1_000_000); a naive
// rand.Read() % 1_000_000 would bias toward low digits because 2^N is not a
// multiple of 1_000_000 — small but real and observable over many samples.
//
// bcrypt with DefaultCost (10) takes ~70ms per Verify, more than enough
// online-friction given the ≤5 attempts cap + 10m TTL.
func GenerateOTP() (code string, hash []byte, err error) {
	n, err := rand.Int(rand.Reader, big.NewInt(otpRange))
	if err != nil {
		return "", nil, fmt.Errorf("auth: read random: %w", err)
	}
	code = fmt.Sprintf("%0*d", otpDigits, n.Int64())
	h, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, fmt.Errorf("auth: bcrypt: %w", err)
	}
	return code, h, nil
}

// VerifyOTP compares a plaintext code against the stored bcrypt hash.
// bcrypt.CompareHashAndPassword is constant-time for its hash compare.
func VerifyOTP(code string, hash []byte) error {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(code)); err != nil {
		return ErrOTPMismatch
	}
	return nil
}
