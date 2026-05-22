package auth

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRefreshToken_UniqueAndCorrectLength(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 1024; i++ {
		raw, hash, err := NewRefreshToken()
		require.NoError(t, err)

		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		require.NoError(t, err)
		assert.Len(t, decoded, refreshTokenBytes)
		assert.Len(t, hash, 32) // sha256 size

		if _, dup := seen[raw]; dup {
			t.Fatalf("duplicate token after %d samples", i)
		}
		seen[raw] = struct{}{}
	}
}

func TestHashRefreshToken_Deterministic(t *testing.T) {
	raw, h1, err := NewRefreshToken()
	require.NoError(t, err)
	h2 := HashRefreshToken(raw)
	assert.Equal(t, h1, h2)
	// Mutating raw changes the hash.
	h3 := HashRefreshToken(raw + "x")
	assert.NotEqual(t, h1, h3)
}
