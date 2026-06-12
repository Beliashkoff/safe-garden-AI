package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	yandexOAuthBaseURL = "https://oauth.yandex.ru"
	yandexLoginBaseURL = "https://login.yandex.ru"
	// yandexJWTIssuer — `iss` claim of login.yandex.ru/info?format=jwt.
	yandexJWTIssuer = "login.yandex.ru"
)

// YandexConfig configures the Yandex ID OAuth client. Base URLs are
// overridable for tests only; production leaves them empty.
type YandexConfig struct {
	ClientID     string
	ClientSecret string
	// RedirectURI must exactly match the Callback URI registered at
	// oauth.yandex.ru. The mobile app catches it via the system browser.
	RedirectURI string

	OAuthBaseURL string
	LoginBaseURL string
	HTTPClient   *http.Client
}

// YandexClient implements the backend half of the Yandex ID authorization-code
// flow (PKCE + client_secret, both server-side): build the authorize URL,
// exchange the code, then fetch the user's identity as a JWT signed with our
// client_secret (HS256) — which cryptographically binds the access token to
// this application, closing the token-substitution hole (Yandex has no
// introspection endpoint).
type YandexClient struct {
	cfg  YandexConfig
	http *http.Client
}

// NewYandex constructs the client. With an empty ClientID the client reports
// !Configured() and SignIn fails fast — dev boots without credentials.
func NewYandex(cfg YandexConfig) *YandexClient {
	if cfg.OAuthBaseURL == "" {
		cfg.OAuthBaseURL = yandexOAuthBaseURL
	}
	if cfg.LoginBaseURL == "" {
		cfg.LoginBaseURL = yandexLoginBaseURL
	}
	h := cfg.HTTPClient
	if h == nil {
		h = &http.Client{Timeout: 10 * time.Second}
	}
	return &YandexClient{cfg: cfg, http: h}
}

// Configured reports whether real credentials are present.
func (c *YandexClient) Configured() bool {
	return c.cfg.ClientID != "" && c.cfg.ClientSecret != "" && c.cfg.RedirectURI != ""
}

// AuthorizeURL builds the oauth.yandex.ru/authorize URL the mobile app opens
// in the system browser. Scopes are fixed to the minimum the product needs
// (profile + email); state and the S256 challenge come from the usecase.
func (c *YandexClient) AuthorizeURL(state, codeChallenge string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", c.cfg.ClientID)
	q.Set("redirect_uri", c.cfg.RedirectURI)
	q.Set("scope", "login:info login:email")
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	return c.cfg.OAuthBaseURL + "/authorize?" + q.Encode()
}

// SignIn exchanges the authorization code and returns the verified identity.
func (c *YandexClient) SignIn(ctx context.Context, code, codeVerifier string) (ExternalIdentity, error) {
	if !c.Configured() {
		return ExternalIdentity{}, fmt.Errorf("auth.Yandex: not configured")
	}
	accessToken, err := c.exchange(ctx, code, codeVerifier)
	if err != nil {
		return ExternalIdentity{}, fmt.Errorf("auth.Yandex: exchange: %w", err)
	}
	id, err := c.fetchIdentity(ctx, accessToken)
	if err != nil {
		return ExternalIdentity{}, fmt.Errorf("auth.Yandex: identity: %w", err)
	}
	return id, nil
}

func (c *YandexClient) exchange(ctx context.Context, code, codeVerifier string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	form.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.OAuthBaseURL+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("post token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return "", fmt.Errorf("%w: status %d (%s)", ErrOAuthRejected, resp.StatusCode, oauthErrorCode(body))
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint status %d", resp.StatusCode)
	}

	var tok struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("empty access_token in response")
	}
	return tok.AccessToken, nil
}

// fetchIdentity calls login.yandex.ru/info?format=jwt and validates the HS256
// signature with our client_secret. A token issued to another application
// cannot produce a JWT that verifies under our secret.
func (c *YandexClient) fetchIdentity(ctx context.Context, accessToken string) (ExternalIdentity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.cfg.LoginBaseURL+"/info?format=jwt", nil)
	if err != nil {
		return ExternalIdentity{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "OAuth "+accessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return ExternalIdentity{}, fmt.Errorf("get info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ExternalIdentity{}, fmt.Errorf("read info response: %w", err)
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return ExternalIdentity{}, fmt.Errorf("%w: info status %d", ErrOAuthRejected, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return ExternalIdentity{}, fmt.Errorf("info endpoint status %d", resp.StatusCode)
	}

	claims := jwt.MapClaims{}
	_, err = jwt.NewParser(
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(yandexJWTIssuer),
		jwt.WithExpirationRequired(),
	).ParseWithClaims(string(body), claims, func(*jwt.Token) (any, error) {
		return []byte(c.cfg.ClientSecret), nil
	})
	if err != nil {
		return ExternalIdentity{}, fmt.Errorf("%w: jwt verify: %v", ErrOAuthRejected, err)
	}

	uid := claimString(claims, "uid")
	if uid == "" {
		return ExternalIdentity{}, fmt.Errorf("%w: jwt missing uid", ErrOAuthRejected)
	}
	return ExternalIdentity{
		Provider: ProviderYandex,
		Subject:  uid,
		Email:    claimString(claims, "email"),
		// The address is the user's own Yandex mailbox, vouched by Yandex.
		EmailVerified: true,
	}, nil
}

// claimString coerces a claim that providers serialize as either a string or
// a JSON number (Yandex uid).
func claimString(claims jwt.MapClaims, key string) string {
	switch v := claims[key].(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

// oauthErrorCode extracts the standard {"error": "..."} code for diagnostics
// without logging the full body (which may echo request material).
func oauthErrorCode(body []byte) string {
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &e); err != nil || e.Error == "" {
		return "unknown_error"
	}
	return e.Error
}
