package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTestKey generates a 2048-bit RSA key and writes it as PKCS#8 PEM
// at <dir>/<kid>.pem. Returns the in-memory key for assertions.
func writeTestKey(t *testing.T, dir, kid string) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	require.NoError(t, os.WriteFile(filepath.Join(dir, kid+".pem"), pemBytes, 0o600))
	return key
}

func newTestIssuer(t *testing.T, dir, activeKID string, now func() time.Time) *Issuer {
	t.Helper()
	iss, err := NewIssuer(IssuerConfig{
		KeysDir:   dir,
		ActiveKID: activeKID,
		AccessTTL: 15 * time.Minute,
		Now:       now,
	})
	require.NoError(t, err)
	return iss
}

func TestIssuer_RoundtripAndClaims(t *testing.T) {
	dir := t.TempDir()
	writeTestKey(t, dir, "k1")
	now := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	iss := newTestIssuer(t, dir, "k1", func() time.Time { return now })

	uid := uuid.New()
	tok, exp, err := iss.Issue(uid)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
	assert.Equal(t, now.Add(15*time.Minute).Unix(), exp.UTC().Unix())

	claims, err := iss.Parse(tok)
	require.NoError(t, err)
	assert.Equal(t, uid, claims.Sub)
	assert.NotEqual(t, uuid.Nil, claims.Jti)
	assert.Equal(t, now.Unix(), claims.Iat.Unix())
	assert.Equal(t, exp.Unix(), claims.Exp.Unix())
}

func TestIssuer_RejectsExpired(t *testing.T) {
	dir := t.TempDir()
	writeTestKey(t, dir, "k1")
	now := time.Now().UTC()
	cur := now
	iss := newTestIssuer(t, dir, "k1", func() time.Time { return cur })

	tok, _, err := iss.Issue(uuid.New())
	require.NoError(t, err)

	// Advance time past TTL + leeway (15m + 60s).
	cur = now.Add(20 * time.Minute)
	_, err = iss.Parse(tok)
	require.Error(t, err)
}

func TestIssuer_RejectsHS256Confusion(t *testing.T) {
	// Classic attack: attacker rewrites alg to HS256 and signs with the
	// server's public key bytes as the HMAC secret. Our parser must refuse
	// because WithValidMethods only allows RS256.
	dir := t.TempDir()
	pk := writeTestKey(t, dir, "k1")
	iss := newTestIssuer(t, dir, "k1", time.Now)

	pubDER, err := x509.MarshalPKIXPublicKey(&pk.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	maliciousClaims := jwt.MapClaims{
		"sub": uuid.New().String(),
		"jti": uuid.New().String(),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	bad := jwt.NewWithClaims(jwt.SigningMethodHS256, maliciousClaims)
	bad.Header["kid"] = "k1"
	signed, err := bad.SignedString(pubPEM)
	require.NoError(t, err)

	_, err = iss.Parse(signed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signing method")
}

func TestIssuer_RejectsNoneAlg(t *testing.T) {
	dir := t.TempDir()
	writeTestKey(t, dir, "k1")
	iss := newTestIssuer(t, dir, "k1", time.Now)

	// Craft an unsigned token with alg=none and the matching kid.
	header := `{"alg":"none","kid":"k1","typ":"JWT"}`
	payload := `{"sub":"` + uuid.New().String() + `","jti":"` + uuid.New().String() + `","iat":1,"exp":9999999999}`
	enc := func(s string) string {
		out := jwtBase64Encode([]byte(s))
		return out
	}
	raw := enc(header) + "." + enc(payload) + "." // empty signature
	_, err := iss.Parse(raw)
	require.Error(t, err)
}

func TestIssuer_RejectsTamperedSignature(t *testing.T) {
	dir := t.TempDir()
	writeTestKey(t, dir, "k1")
	iss := newTestIssuer(t, dir, "k1", time.Now)
	tok, _, err := iss.Issue(uuid.New())
	require.NoError(t, err)

	parts := splitJWT(tok)
	require.Len(t, parts, 3)
	// Flip the last byte of the signature.
	sig := []byte(parts[2])
	sig[len(sig)-1] ^= 0xFF
	tampered := parts[0] + "." + parts[1] + "." + string(sig)

	_, err = iss.Parse(tampered)
	require.Error(t, err)
}

func TestIssuer_KidRotationGrace(t *testing.T) {
	// Old kid "k1" should still verify after we promote "k2" to active —
	// otherwise rotation breaks every in-flight 15-minute access token.
	dir := t.TempDir()
	writeTestKey(t, dir, "k1")
	writeTestKey(t, dir, "k2")

	now := time.Now().UTC()
	cur := now
	issOld := newTestIssuer(t, dir, "k1", func() time.Time { return cur })
	tokOld, _, err := issOld.Issue(uuid.New())
	require.NoError(t, err)

	issNew := newTestIssuer(t, dir, "k2", func() time.Time { return cur })
	assert.Equal(t, "k2", issNew.ActiveKID())

	_, err = issNew.Parse(tokOld)
	require.NoError(t, err)
}

func TestIssuer_RejectsUnknownKid(t *testing.T) {
	dir := t.TempDir()
	writeTestKey(t, dir, "k1")
	iss := newTestIssuer(t, dir, "k1", time.Now)

	// Manually sign a valid-looking RS256 token with kid=unknown using a key
	// the issuer doesn't have. We just need to confirm the keyfunc rejects
	// before checking the signature.
	outsideKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	other := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": uuid.New().String(),
		"jti": uuid.New().String(),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	other.Header["kid"] = "k-other"
	signed, err := other.SignedString(outsideKey)
	require.NoError(t, err)

	_, err = iss.Parse(signed)
	require.Error(t, err)
}

func TestNewIssuer_SingleFileFallback(t *testing.T) {
	dir := t.TempDir()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	path := filepath.Join(dir, "single.pem")
	require.NoError(t, os.WriteFile(path,
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600))

	iss, err := NewIssuer(IssuerConfig{
		PrivateKeyPath: path,
		KID:            "solo",
		AccessTTL:      time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, "solo", iss.ActiveKID())

	tok, _, err := iss.Issue(uuid.New())
	require.NoError(t, err)
	_, err = iss.Parse(tok)
	require.NoError(t, err)
}

func TestNewIssuer_RejectsMissingConfig(t *testing.T) {
	_, err := NewIssuer(IssuerConfig{AccessTTL: time.Minute})
	require.Error(t, err)
	_, err = NewIssuer(IssuerConfig{KeysDir: t.TempDir(), AccessTTL: time.Minute})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ActiveKID")
	_, err = NewIssuer(IssuerConfig{KeysDir: t.TempDir(), ActiveKID: "nope", AccessTTL: time.Minute})
	require.Error(t, err)
}

func TestNewIssuer_RejectsActiveKIDNotInDir(t *testing.T) {
	dir := t.TempDir()
	writeTestKey(t, dir, "k1")
	_, err := NewIssuer(IssuerConfig{KeysDir: dir, ActiveKID: "missing", AccessTTL: time.Minute})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "missing") || errors.Is(err, errors.New("not loaded")))
}

// --- small helpers for the none-alg and tamper tests -------------------------

func splitJWT(s string) []string { return strings.Split(s, ".") }

// jwtBase64Encode encodes without padding, matching JWT (base64url).
func jwtBase64Encode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
