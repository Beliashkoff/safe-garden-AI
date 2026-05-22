package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

const (
	appleIssuer  = "https://appleid.apple.com"
	googleIssuer = "https://accounts.google.com"
)

// ExternalIdentity is what we trust after a successful OIDC verification.
//
// Note on Apple: Subject (apple_sub) is the only stable identity. Email may
// be a private-relay value and is treated as informational, never as the
// identity key. EmailVerified is honoured for Google but the caller may
// still choose to re-verify via OTP.
type ExternalIdentity struct {
	Provider      string
	Subject       string
	Email         string
	EmailVerified bool
}

// VerifierConfig configures provider audiences.
type VerifierConfig struct {
	AppleBundleID    string
	GoogleClientIOS  string
	GoogleClientAndr string

	// Optional overrides for tests — when non-empty, used in place of the
	// real Apple/Google issuer URLs. Production code leaves these empty.
	AppleIssuerOverride  string
	GoogleIssuerOverride string

	// DiscoveryTimeout caps each provider's well-known/openid-configuration
	// fetch. Default is 10s.
	DiscoveryTimeout time.Duration
}

// Verifier verifies Apple and Google id_tokens.
type Verifier struct {
	appleVerifier  *oidc.IDTokenVerifier
	googleVerifier *oidc.IDTokenVerifier
	googleClients  map[string]struct{}
}

// NewVerifier constructs a Verifier and performs the network-dependent JWKS
// discovery for both providers. Production callers should panic-on-error at
// startup if this fails. In dev with empty config, Apple/Google verifiers
// are nil and the corresponding VerifyApple/VerifyGoogle calls will return
// a clear error — boot is not blocked.
func NewVerifier(ctx context.Context, cfg VerifierConfig) (*Verifier, error) {
	timeout := cfg.DiscoveryTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	v := &Verifier{googleClients: map[string]struct{}{}}
	if cfg.GoogleClientIOS != "" {
		v.googleClients[cfg.GoogleClientIOS] = struct{}{}
	}
	if cfg.GoogleClientAndr != "" {
		v.googleClients[cfg.GoogleClientAndr] = struct{}{}
	}

	if cfg.AppleBundleID != "" {
		issuer := appleIssuer
		if cfg.AppleIssuerOverride != "" {
			issuer = cfg.AppleIssuerOverride
		}
		dctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		ap, err := oidc.NewProvider(dctx, issuer)
		if err != nil {
			return nil, fmt.Errorf("auth.NewVerifier: apple provider: %w", err)
		}
		v.appleVerifier = ap.Verifier(&oidc.Config{
			ClientID:             cfg.AppleBundleID,
			SupportedSigningAlgs: []string{"RS256"},
		})
	}

	if len(v.googleClients) > 0 {
		issuer := googleIssuer
		if cfg.GoogleIssuerOverride != "" {
			issuer = cfg.GoogleIssuerOverride
		}
		dctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		gp, err := oidc.NewProvider(dctx, issuer)
		if err != nil {
			return nil, fmt.Errorf("auth.NewVerifier: google provider: %w", err)
		}
		v.googleVerifier = gp.Verifier(&oidc.Config{
			SkipClientIDCheck:    true, // multi-aud allowlist enforced manually
			SupportedSigningAlgs: []string{"RS256"},
		})
	}

	return v, nil
}

// VerifyApple validates an Apple id_token and the caller-supplied nonce.
//
// The nonce flow assumed here is the modern Apple Sign-In one: the mobile
// app generates rawNonce, sends sha256(rawNonce) to Apple as the `nonce`
// parameter, and Apple echoes the rawNonce in the id_token's `nonce` claim
// (because nonce_supported=true on iOS). The mobile app POSTs both the
// id_token and rawNonce to us; we compare against the claim in constant time.
//
// We never trust Apple's email as identity (apple_sub only); email may be a
// private-relay address and is returned only for informational use.
func (v *Verifier) VerifyApple(ctx context.Context, rawIDToken, expectedNonce string) (ExternalIdentity, error) {
	if v.appleVerifier == nil {
		return ExternalIdentity{}, errors.New("auth.VerifyApple: not configured")
	}
	if expectedNonce == "" {
		return ExternalIdentity{}, errors.New("auth.VerifyApple: nonce required")
	}
	tok, err := v.appleVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return ExternalIdentity{}, fmt.Errorf("auth.VerifyApple: %w", err)
	}
	var c struct {
		Email         string `json:"email"`
		EmailVerified any    `json:"email_verified"` // Apple returns "true"/"false" as string sometimes
		Nonce         string `json:"nonce"`
	}
	if err := tok.Claims(&c); err != nil {
		return ExternalIdentity{}, fmt.Errorf("auth.VerifyApple: claims: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(c.Nonce), []byte(expectedNonce)) != 1 {
		return ExternalIdentity{}, errors.New("auth.VerifyApple: nonce mismatch")
	}
	return ExternalIdentity{
		Provider:      "apple",
		Subject:       tok.Subject,
		Email:         c.Email,
		EmailVerified: parseFlexibleBool(c.EmailVerified),
	}, nil
}

// VerifyGoogle validates a Google id_token against the explicit audience
// allowlist (iOS + Android client IDs).
func (v *Verifier) VerifyGoogle(ctx context.Context, rawIDToken string) (ExternalIdentity, error) {
	if v.googleVerifier == nil {
		return ExternalIdentity{}, errors.New("auth.VerifyGoogle: not configured")
	}
	tok, err := v.googleVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return ExternalIdentity{}, fmt.Errorf("auth.VerifyGoogle: %w", err)
	}
	if !v.audienceAllowed(tok.Audience) {
		return ExternalIdentity{}, fmt.Errorf("auth.VerifyGoogle: audience %v not allowed", tok.Audience)
	}
	var c struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := tok.Claims(&c); err != nil {
		return ExternalIdentity{}, fmt.Errorf("auth.VerifyGoogle: claims: %w", err)
	}
	return ExternalIdentity{
		Provider:      "google",
		Subject:       tok.Subject,
		Email:         c.Email,
		EmailVerified: c.EmailVerified,
	}, nil
}

// audienceAllowed requires every audience entry to be in the allowlist. This
// is stricter than "any match" — a token issued for an unknown extra aud is
// rejected even if one of its auds is known.
func (v *Verifier) audienceAllowed(aud []string) bool {
	if len(aud) == 0 {
		return false
	}
	for _, a := range aud {
		if _, ok := v.googleClients[a]; !ok {
			return false
		}
	}
	return true
}

// parseFlexibleBool handles bool, "true"/"false" string, and numeric variants
// — Apple has historically returned email_verified inconsistently.
func parseFlexibleBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		b, err := strconv.ParseBool(x)
		return err == nil && b
	default:
		return false
	}
}
