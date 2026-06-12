package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidEmail(t *testing.T) {
	cases := map[string]bool{
		"user@example.com":        true,
		"a.b+tag@sub.example.co":  true,
		"":                        false,
		"no-at-sign":              false,
		"two@@example.com":        false,
		"name <user@example.com>": false, // bare address only
	}
	for in, want := range cases {
		assert.Equalf(t, want, validEmail(in), "validEmail(%q)", in)
	}
}

func TestValidOTPFormat(t *testing.T) {
	assert.True(t, validOTPFormat("000000"))
	assert.True(t, validOTPFormat("123456"))
	assert.False(t, validOTPFormat("12345"))   // too short
	assert.False(t, validOTPFormat("1234567")) // too long
	assert.False(t, validOTPFormat("12a456"))  // non-digit
	assert.False(t, validOTPFormat(""))
}

func TestNormalizeEmail(t *testing.T) {
	assert.Equal(t, "user@example.com", normalizeEmail("  User@Example.COM "))
}

func TestCanLinkByEmail(t *testing.T) {
	// Provider-verified address (Yandex) links to an existing account.
	assert.True(t, canLinkByEmail("user@example.com", true))
	// Unverified address (VK — no confirmation guarantee) never links.
	assert.False(t, canLinkByEmail("user@example.com", false))
	// Empty never links.
	assert.False(t, canLinkByEmail("", true))
}

func TestEmailForStorage(t *testing.T) {
	// Verified (Yandex) email is stored and marked verified.
	store, verified := emailForStorage("user@yandex.ru", true)
	assert.Equal(t, "user@yandex.ru", store)
	assert.True(t, verified)

	// Unverified (VK) email is dropped to avoid squatting the unique slot.
	store, verified = emailForStorage("user@example.com", false)
	assert.Equal(t, "", store)
	assert.False(t, verified)

	// No email.
	store, verified = emailForStorage("", true)
	assert.Equal(t, "", store)
	assert.False(t, verified)
}
