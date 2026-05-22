// Package auth provides authentication primitives: JWT issuance/verification
// with kid rotation, OIDC verification for Apple/Google, opaque refresh tokens,
// and email OTP. Stage 1.1 covers the primitives; HTTP wiring lives in stage 1.2.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// refreshTokenBytes — 32 bytes of entropy (256 bits). At this size the only
// realistic threat is online presentation, which is gated by rate limits and
// reuse-detection downstream.
const refreshTokenBytes = 32

// NewRefreshToken generates a fresh opaque refresh token. The raw value is
// returned to the client and never persisted; the sha256 hash is stored in
// refresh_tokens.token_hash with a UNIQUE index.
//
// Storing only the hash means a database leak does not yield usable tokens.
// The raw token has 256 bits of entropy, so timing-side-channels on hash
// lookup are not exploitable (the attacker cannot produce a candidate to
// compare against).
func NewRefreshToken() (raw string, hash []byte, err error) {
	b := make([]byte, refreshTokenBytes)
	if _, err = rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("auth: read random: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(raw))
	return raw, h[:], nil
}

// HashRefreshToken returns sha256(raw) for DB lookup. Deterministic.
func HashRefreshToken(raw string) []byte {
	h := sha256.Sum256([]byte(raw))
	return h[:]
}
