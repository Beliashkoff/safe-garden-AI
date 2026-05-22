package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIDP hosts /.well-known/openid-configuration + /jwks and lets tests
// mint id_tokens with a controlled set of claims.
type fakeIDP struct {
	srv    *httptest.Server
	key    *rsa.PrivateKey
	kid    string
	issuer string
}

func newFakeIDP(t *testing.T) *fakeIDP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	idp := &fakeIDP{key: key, kid: "test-key"}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"issuer":"%s",
			"jwks_uri":"%s/jwks",
			"authorization_endpoint":"%s/auth",
			"token_endpoint":"%s/token",
			"response_types_supported":["id_token"],
			"subject_types_supported":["public"],
			"id_token_signing_alg_values_supported":["RS256"]
		}`, idp.issuer, idp.issuer, idp.issuer, idp.issuer)
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
		fmt.Fprintf(w, `{"keys":[{"kty":"RSA","use":"sig","alg":"RS256","kid":"%s","n":"%s","e":"%s"}]}`,
			idp.kid, n, e)
	})

	idp.srv = httptest.NewServer(mux)
	idp.issuer = idp.srv.URL
	return idp
}

func (idp *fakeIDP) Close() { idp.srv.Close() }

// sign produces an RS256 id_token with the given claims and kid header.
func (idp *fakeIDP) sign(t *testing.T, claims jwt.MapClaims, alg jwt.SigningMethod, signKey any) string {
	t.Helper()
	if alg == nil {
		alg = jwt.SigningMethodRS256
	}
	if signKey == nil {
		signKey = idp.key
	}
	tok := jwt.NewWithClaims(alg, claims)
	tok.Header["kid"] = idp.kid
	out, err := tok.SignedString(signKey)
	require.NoError(t, err)
	return out
}

func baseAppleClaims(aud, sub, nonceRaw string, issuer string, exp time.Time) jwt.MapClaims {
	// Apple in the modern flow stores rawNonce in the `nonce` claim.
	return jwt.MapClaims{
		"iss":            issuer,
		"aud":            aud,
		"sub":            sub,
		"iat":            time.Now().Add(-1 * time.Minute).Unix(),
		"exp":            exp.Unix(),
		"email":          "user@example.com",
		"email_verified": "true",
		"nonce":          nonceRaw,
	}
}

func baseGoogleClaims(aud, sub, issuer string, exp time.Time) jwt.MapClaims {
	return jwt.MapClaims{
		"iss":            issuer,
		"aud":            aud,
		"sub":            sub,
		"iat":            time.Now().Add(-1 * time.Minute).Unix(),
		"exp":            exp.Unix(),
		"email":          "user@example.com",
		"email_verified": true,
	}
}

// ------------------ Apple tests ----------------------------------------------

func TestVerifyApple_HappyPath(t *testing.T) {
	idp := newFakeIDP(t)
	defer idp.Close()

	v, err := NewVerifier(context.Background(), VerifierConfig{
		AppleBundleID:       "com.example.app",
		AppleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	nonce := "raw-nonce-abc"
	tok := idp.sign(t, baseAppleClaims("com.example.app", "apple-sub-123", nonce, idp.issuer, time.Now().Add(10*time.Minute)), nil, nil)

	id, err := v.VerifyApple(context.Background(), tok, nonce)
	require.NoError(t, err)
	assert.Equal(t, "apple", id.Provider)
	assert.Equal(t, "apple-sub-123", id.Subject)
	assert.Equal(t, "user@example.com", id.Email)
	assert.True(t, id.EmailVerified)
}

func TestVerifyApple_RejectsEmptyNonce(t *testing.T) {
	idp := newFakeIDP(t)
	defer idp.Close()
	v, err := NewVerifier(context.Background(), VerifierConfig{
		AppleBundleID:       "com.example.app",
		AppleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	tok := idp.sign(t, baseAppleClaims("com.example.app", "sub", "n", idp.issuer, time.Now().Add(10*time.Minute)), nil, nil)
	_, err = v.VerifyApple(context.Background(), tok, "")
	require.Error(t, err)
}

func TestVerifyApple_RejectsNonceMismatch(t *testing.T) {
	idp := newFakeIDP(t)
	defer idp.Close()
	v, err := NewVerifier(context.Background(), VerifierConfig{
		AppleBundleID:       "com.example.app",
		AppleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	tok := idp.sign(t, baseAppleClaims("com.example.app", "sub", "expected", idp.issuer, time.Now().Add(10*time.Minute)), nil, nil)
	_, err = v.VerifyApple(context.Background(), tok, "WRONG")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonce")
}

func TestVerifyApple_RejectsWrongAudience(t *testing.T) {
	idp := newFakeIDP(t)
	defer idp.Close()
	v, err := NewVerifier(context.Background(), VerifierConfig{
		AppleBundleID:       "com.example.app",
		AppleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	tok := idp.sign(t, baseAppleClaims("com.other.app", "sub", "n", idp.issuer, time.Now().Add(10*time.Minute)), nil, nil)
	_, err = v.VerifyApple(context.Background(), tok, "n")
	require.Error(t, err)
}

func TestVerifyApple_RejectsExpired(t *testing.T) {
	idp := newFakeIDP(t)
	defer idp.Close()
	v, err := NewVerifier(context.Background(), VerifierConfig{
		AppleBundleID:       "com.example.app",
		AppleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	tok := idp.sign(t, baseAppleClaims("com.example.app", "sub", "n", idp.issuer, time.Now().Add(-1*time.Minute)), nil, nil)
	_, err = v.VerifyApple(context.Background(), tok, "n")
	require.Error(t, err)
}

func TestVerifyApple_RejectsHS256(t *testing.T) {
	idp := newFakeIDP(t)
	defer idp.Close()
	v, err := NewVerifier(context.Background(), VerifierConfig{
		AppleBundleID:       "com.example.app",
		AppleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	// Sign with HS256 using a symmetric secret — verifier should refuse the alg.
	tok := idp.sign(t, baseAppleClaims("com.example.app", "sub", "n", idp.issuer, time.Now().Add(10*time.Minute)),
		jwt.SigningMethodHS256, []byte("symmetric-secret"))
	_, err = v.VerifyApple(context.Background(), tok, "n")
	require.Error(t, err)
}

func TestVerifyApple_NotConfigured(t *testing.T) {
	v, err := NewVerifier(context.Background(), VerifierConfig{})
	require.NoError(t, err)
	_, err = v.VerifyApple(context.Background(), "irrelevant", "n")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestVerifyApple_EmailVerifiedFalseString(t *testing.T) {
	idp := newFakeIDP(t)
	defer idp.Close()
	v, err := NewVerifier(context.Background(), VerifierConfig{
		AppleBundleID:       "com.example.app",
		AppleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	claims := baseAppleClaims("com.example.app", "sub", "n", idp.issuer, time.Now().Add(10*time.Minute))
	claims["email_verified"] = "false"
	tok := idp.sign(t, claims, nil, nil)

	id, err := v.VerifyApple(context.Background(), tok, "n")
	require.NoError(t, err)
	assert.False(t, id.EmailVerified)
}

// ------------------ Google tests ---------------------------------------------

func TestVerifyGoogle_HappyPath_iOS(t *testing.T) {
	idp := newFakeIDP(t)
	defer idp.Close()
	v, err := NewVerifier(context.Background(), VerifierConfig{
		GoogleClientIOS:      "ios.apps.googleusercontent.com",
		GoogleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	tok := idp.sign(t, baseGoogleClaims("ios.apps.googleusercontent.com", "google-sub-1", idp.issuer, time.Now().Add(10*time.Minute)), nil, nil)
	id, err := v.VerifyGoogle(context.Background(), tok)
	require.NoError(t, err)
	assert.Equal(t, "google", id.Provider)
	assert.Equal(t, "google-sub-1", id.Subject)
	assert.True(t, id.EmailVerified)
}

func TestVerifyGoogle_RejectsAudNotInAllowlist(t *testing.T) {
	idp := newFakeIDP(t)
	defer idp.Close()
	v, err := NewVerifier(context.Background(), VerifierConfig{
		GoogleClientIOS:      "ios.apps.googleusercontent.com",
		GoogleClientAndr:     "android.apps.googleusercontent.com",
		GoogleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	tok := idp.sign(t, baseGoogleClaims("attacker.apps.googleusercontent.com", "sub", idp.issuer, time.Now().Add(10*time.Minute)), nil, nil)
	_, err = v.VerifyGoogle(context.Background(), tok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audience")
}

func TestVerifyGoogle_RejectsExpired(t *testing.T) {
	idp := newFakeIDP(t)
	defer idp.Close()
	v, err := NewVerifier(context.Background(), VerifierConfig{
		GoogleClientAndr:     "android.apps.googleusercontent.com",
		GoogleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	tok := idp.sign(t, baseGoogleClaims("android.apps.googleusercontent.com", "sub", idp.issuer, time.Now().Add(-1*time.Minute)), nil, nil)
	_, err = v.VerifyGoogle(context.Background(), tok)
	require.Error(t, err)
}

func TestVerifyGoogle_RejectsHS256(t *testing.T) {
	idp := newFakeIDP(t)
	defer idp.Close()
	v, err := NewVerifier(context.Background(), VerifierConfig{
		GoogleClientIOS:      "ios.apps.googleusercontent.com",
		GoogleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	tok := idp.sign(t, baseGoogleClaims("ios.apps.googleusercontent.com", "sub", idp.issuer, time.Now().Add(10*time.Minute)),
		jwt.SigningMethodHS256, []byte("symmetric"))
	_, err = v.VerifyGoogle(context.Background(), tok)
	require.Error(t, err)
}

func TestVerifyGoogle_NotConfigured(t *testing.T) {
	v, err := NewVerifier(context.Background(), VerifierConfig{})
	require.NoError(t, err)
	_, err = v.VerifyGoogle(context.Background(), "irrelevant")
	require.Error(t, err)
}

func TestNewVerifier_FailsOnUnreachableIssuer(t *testing.T) {
	// Use a server URL we immediately close — the dial fails fast.
	idp := newFakeIDP(t)
	url := idp.issuer
	idp.Close()

	_, err := NewVerifier(context.Background(), VerifierConfig{
		AppleBundleID:       "com.example.app",
		AppleIssuerOverride: url,
		DiscoveryTimeout:    2 * time.Second,
	})
	require.Error(t, err)
}

// ------------------ misc -----------------------------------------------------

func TestParseFlexibleBool(t *testing.T) {
	cases := []struct {
		in   any
		want bool
	}{
		{true, true},
		{false, false},
		{"true", true},
		{"false", false},
		{"True", true},
		{"yes", false}, // strconv.ParseBool rejects "yes"
		{nil, false},
		{42, false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, parseFlexibleBool(c.in), fmt.Sprintf("input %#v", c.in))
	}
}
