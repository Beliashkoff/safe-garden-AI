package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidEmail(t *testing.T) {
	cases := map[string]bool{
		"user@example.com":              true,
		"a.b+tag@sub.example.co":        true,
		"":                              false,
		"no-at-sign":                    false,
		"two@@example.com":              false,
		"name <user@example.com>":       false, // bare address only
		"user@privaterelay.appleid.com": true,
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

func TestIsApplePrivateRelay(t *testing.T) {
	assert.True(t, isApplePrivateRelay("abc123@privaterelay.appleid.com"))
	assert.False(t, isApplePrivateRelay("user@example.com"))
}

func TestCanLinkByEmail(t *testing.T) {
	// Apple: real address links (provider-verified), relay does not.
	assert.True(t, canLinkByEmail("apple", "user@example.com", false))
	assert.False(t, canLinkByEmail("apple", "x@privaterelay.appleid.com", true))
	// Google: only when email_verified.
	assert.True(t, canLinkByEmail("google", "user@example.com", true))
	assert.False(t, canLinkByEmail("google", "user@example.com", false))
	// Empty never links.
	assert.False(t, canLinkByEmail("google", "", true))
}

func TestEmailForStorage(t *testing.T) {
	store, verified := emailForStorage("apple", "user@example.com", false)
	assert.Equal(t, "user@example.com", store)
	assert.True(t, verified)

	// Relay: stored and treated as verified/deliverable.
	store, verified = emailForStorage("apple", "x@privaterelay.appleid.com", false)
	assert.Equal(t, "x@privaterelay.appleid.com", store)
	assert.True(t, verified)

	// Unverified Google email is dropped to avoid collisions/impersonation.
	store, verified = emailForStorage("google", "user@example.com", false)
	assert.Equal(t, "", store)
	assert.False(t, verified)

	// No email.
	store, verified = emailForStorage("google", "", true)
	assert.Equal(t, "", store)
	assert.False(t, verified)
}
