package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCE + state material for the OAuth authorization-code flows (Yandex ID,
// VK ID). The code_verifier never leaves the backend: the mobile client only
// receives the derived S256 challenge (and, for Yandex, the full authorize
// URL). 32 bytes of entropy → 43 base64url chars, which satisfies both RFC
// 7636 (43–128 chars) and VK ID's state requirement (≥32 chars of [A-Za-z0-9_-]).
const oauthRandomBytes = 32

// NewOAuthState generates the CSRF state value and its sha256 hash. Only the
// hash is persisted (oauth_states.state_hash) so a DB leak does not yield
// replayable states.
func NewOAuthState() (raw string, hash []byte, err error) {
	raw, err = randomURLSafe()
	if err != nil {
		return "", nil, fmt.Errorf("auth: new state: %w", err)
	}
	return raw, HashOAuthState(raw), nil
}

// HashOAuthState returns sha256(raw) for DB lookup. Deterministic.
func HashOAuthState(raw string) []byte {
	h := sha256.Sum256([]byte(raw))
	return h[:]
}

// NewPKCEVerifier generates a code_verifier and its S256 code_challenge
// (RFC 7636 §4.2: base64url-no-padding of sha256(verifier)).
func NewPKCEVerifier() (verifier, challenge string, err error) {
	verifier, err = randomURLSafe()
	if err != nil {
		return "", "", fmt.Errorf("auth: new verifier: %w", err)
	}
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

func randomURLSafe() (string, error) {
	b := make([]byte, oauthRandomBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
