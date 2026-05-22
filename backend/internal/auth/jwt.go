package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the access-token payload.
type Claims struct {
	Sub uuid.UUID
	Jti uuid.UUID
	Iat time.Time
	Exp time.Time
}

// IssuerConfig configures the RS256 token issuer. Provide either
// KeysDir+ActiveKID (multi-key, recommended for prod) or
// PrivateKeyPath+KID (single-key, dev convenience).
type IssuerConfig struct {
	KeysDir        string
	ActiveKID      string
	PrivateKeyPath string
	KID            string
	AccessTTL      time.Duration
	Now            func() time.Time // optional; defaults to time.Now
}

// Issuer signs JWTs with one active key and verifies tokens against any
// loaded key looked up by header "kid". Multiple keys live in memory at once
// so a rotation can swap the active kid without invalidating in-flight
// tokens — they remain valid until natural expiry (15m).
type Issuer struct {
	signKID    string
	signKey    *rsa.PrivateKey
	verifyKeys map[string]*rsa.PublicKey
	ttl        time.Duration
	now        func() time.Time
}

// NewIssuer loads all RSA private keys per cfg and constructs an Issuer.
func NewIssuer(cfg IssuerConfig) (*Issuer, error) {
	if cfg.AccessTTL <= 0 {
		return nil, errors.New("auth.NewIssuer: AccessTTL must be > 0")
	}
	keys, activeKID, err := loadKeys(cfg)
	if err != nil {
		return nil, err
	}
	active, ok := keys[activeKID]
	if !ok {
		return nil, fmt.Errorf("auth.NewIssuer: active kid %q not loaded", activeKID)
	}
	verify := make(map[string]*rsa.PublicKey, len(keys))
	for kid, k := range keys {
		verify[kid] = &k.PublicKey
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Issuer{
		signKID:    activeKID,
		signKey:    active,
		verifyKeys: verify,
		ttl:        cfg.AccessTTL,
		now:        now,
	}, nil
}

// Issue produces a signed RS256 token for userID. The returned exp is the
// absolute expiration time (UTC).
func (i *Issuer) Issue(userID uuid.UUID) (token string, exp time.Time, err error) {
	jti, err := uuid.NewRandom()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth.Issue: jti: %w", err)
	}
	now := i.now().UTC()
	exp = now.Add(i.ttl)
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"jti": jti.String(),
		"iat": now.Unix(),
		"exp": exp.Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	t.Header["kid"] = i.signKID
	signed, err := t.SignedString(i.signKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth.Issue: sign: %w", err)
	}
	return signed, exp, nil
}

// Parse verifies the token and returns the structured claims.
//
// Security hardening lives in the parser options:
//   - WithValidMethods(["RS256"]) — refuses alg=none and the HS256-confusion
//     attack (where the public key is fed to HMAC as the secret).
//   - WithExpirationRequired/WithIssuedAt — refuses tokens missing exp/iat.
//   - WithLeeway(60s) — small skew tolerance, not a free pass to past expiry.
//
// The keyfunc requires a "kid" header and refuses unknown kids.
func (i *Issuer) Parse(raw string) (Claims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithLeeway(60*time.Second),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(func() time.Time { return i.now() }),
	)
	mc := jwt.MapClaims{}
	tok, err := parser.ParseWithClaims(raw, mc, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing kid")
		}
		pub, ok := i.verifyKeys[kid]
		if !ok {
			return nil, fmt.Errorf("unknown kid %q", kid)
		}
		return pub, nil
	})
	if err != nil {
		return Claims{}, fmt.Errorf("auth.Parse: %w", err)
	}
	if !tok.Valid {
		return Claims{}, errors.New("auth.Parse: token invalid")
	}
	sub, err := uuidFromClaim(mc, "sub")
	if err != nil {
		return Claims{}, err
	}
	jti, err := uuidFromClaim(mc, "jti")
	if err != nil {
		return Claims{}, err
	}
	iat, err := timeFromClaim(mc, "iat")
	if err != nil {
		return Claims{}, err
	}
	exp, err := timeFromClaim(mc, "exp")
	if err != nil {
		return Claims{}, err
	}
	return Claims{Sub: sub, Jti: jti, Iat: iat, Exp: exp}, nil
}

// ActiveKID exposes the current signing kid — useful for telemetry.
func (i *Issuer) ActiveKID() string { return i.signKID }

func uuidFromClaim(mc jwt.MapClaims, key string) (uuid.UUID, error) {
	v, ok := mc[key].(string)
	if !ok || v == "" {
		return uuid.Nil, fmt.Errorf("auth.Parse: missing %s", key)
	}
	u, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil, fmt.Errorf("auth.Parse: bad %s: %w", key, err)
	}
	return u, nil
}

func timeFromClaim(mc jwt.MapClaims, key string) (time.Time, error) {
	switch v := mc[key].(type) {
	case float64:
		return time.Unix(int64(v), 0).UTC(), nil
	case int64:
		return time.Unix(v, 0).UTC(), nil
	default:
		return time.Time{}, fmt.Errorf("auth.Parse: bad %s type %T", key, v)
	}
}

// loadKeys reads RSA private keys per cfg. Returns map[kid]*rsa.PrivateKey
// and the resolved active kid (single-file mode sets activeKID = cfg.KID).
func loadKeys(cfg IssuerConfig) (map[string]*rsa.PrivateKey, string, error) {
	keys := map[string]*rsa.PrivateKey{}
	switch {
	case cfg.KeysDir != "":
		if cfg.ActiveKID == "" {
			return nil, "", errors.New("auth.NewIssuer: ActiveKID required with KeysDir")
		}
		entries, err := os.ReadDir(cfg.KeysDir)
		if err != nil {
			return nil, "", fmt.Errorf("auth.NewIssuer: read dir: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".pem") {
				continue
			}
			kid := strings.TrimSuffix(e.Name(), ".pem")
			path := filepath.Join(cfg.KeysDir, e.Name())
			pemBytes, err := os.ReadFile(path) //nolint:gosec // path is under operator-controlled config dir
			if err != nil {
				return nil, "", fmt.Errorf("auth.NewIssuer: read %s: %w", path, err)
			}
			pk, err := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
			if err != nil {
				return nil, "", fmt.Errorf("auth.NewIssuer: parse %s: %w", path, err)
			}
			keys[kid] = pk
		}
		if len(keys) == 0 {
			return nil, "", fmt.Errorf("auth.NewIssuer: no *.pem keys in %s", cfg.KeysDir)
		}
		return keys, cfg.ActiveKID, nil

	case cfg.PrivateKeyPath != "":
		if cfg.KID == "" {
			return nil, "", errors.New("auth.NewIssuer: KID required with PrivateKeyPath")
		}
		pemBytes, err := os.ReadFile(cfg.PrivateKeyPath) //nolint:gosec // operator-controlled
		if err != nil {
			return nil, "", fmt.Errorf("auth.NewIssuer: read key: %w", err)
		}
		pk, err := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
		if err != nil {
			return nil, "", fmt.Errorf("auth.NewIssuer: parse key: %w", err)
		}
		keys[cfg.KID] = pk
		return keys, cfg.KID, nil

	default:
		return nil, "", errors.New("auth.NewIssuer: provide KeysDir or PrivateKeyPath")
	}
}
