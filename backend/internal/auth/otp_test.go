package auth

import (
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateOTP_ShapeAndVerify(t *testing.T) {
	seen := map[string]int{}
	for i := 0; i < 200; i++ {
		code, hash, err := GenerateOTP()
		require.NoError(t, err)
		assert.Len(t, code, otpDigits)

		// Code must be a 6-digit number (no letters, no signs).
		n, err := strconv.Atoi(code)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, n, 0)
		assert.Less(t, n, otpRange)

		// Verify happy path.
		require.NoError(t, VerifyOTP(code, hash))
		// Verify rejects a different code.
		other := "000000"
		if code == other {
			other = "999999"
		}
		err = VerifyOTP(other, hash)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrOTPMismatch))

		seen[code]++
	}
	// Spot-check that we are not stuck on a single value — at least 5 unique codes.
	assert.GreaterOrEqual(t, len(seen), 5)
}

func TestVerifyOTP_GarbageHashRejected(t *testing.T) {
	err := VerifyOTP("123456", []byte("not-a-bcrypt-hash"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOTPMismatch))
}
