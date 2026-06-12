package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// VK ID requires state ≥32 chars of [A-Za-z0-9_-]; RFC 7636 requires the
// verifier to be 43–128 chars of the same alphabet.
var urlSafeRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func TestNewOAuthState(t *testing.T) {
	raw, hash, err := NewOAuthState()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(raw), 32)
	assert.Regexp(t, urlSafeRe, raw)
	assert.Equal(t, HashOAuthState(raw), hash)
	assert.Len(t, hash, sha256.Size)

	raw2, _, err := NewOAuthState()
	require.NoError(t, err)
	assert.NotEqual(t, raw, raw2)
}

func TestNewPKCEVerifier(t *testing.T) {
	verifier, challenge, err := NewPKCEVerifier()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(verifier), 43)
	assert.LessOrEqual(t, len(verifier), 128)
	assert.Regexp(t, urlSafeRe, verifier)

	sum := sha256.Sum256([]byte(verifier))
	assert.Equal(t, base64.RawURLEncoding.EncodeToString(sum[:]), challenge)
}
