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
)

// vkBaseURL — canonical VK ID OAuth host per current docs (id.vk.ru; the
// older id.vk.com remains an alias). Overridable via config for tests.
const vkBaseURL = "https://id.vk.ru"

// VKConfig configures the VK ID OAuth 2.1 client.
type VKConfig struct {
	// ClientID — numeric application ID from the VK ID business cabinet.
	ClientID string
	// RedirectURI must match the deep link the mobile VK ID SDK is configured
	// with (vk{client_id}://vk.ru) and the trusted redirect in the cabinet.
	RedirectURI string

	BaseURL    string
	HTTPClient *http.Client
}

// VKClient implements the backend half of the VK ID authorization-code flow.
// The mobile SDK runs in "confidential" mode: it only surfaces the auth code
// and device_id, while the PKCE verifier is generated and kept here. Because
// this backend performs the /oauth2/auth exchange itself with our client_id
// and verifier, the resulting tokens are provably issued to this application —
// no token-substitution window exists. Provider tokens are used once (to read
// the profile) and discarded; we never persist them (data minimization).
type VKClient struct {
	cfg  VKConfig
	http *http.Client
}

// NewVK constructs the client. With an empty ClientID the client reports
// !Configured() and SignIn fails fast — dev boots without credentials.
func NewVK(cfg VKConfig) *VKClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = vkBaseURL
	}
	h := cfg.HTTPClient
	if h == nil {
		h = &http.Client{Timeout: 10 * time.Second}
	}
	return &VKClient{cfg: cfg, http: h}
}

// Configured reports whether real credentials are present.
func (c *VKClient) Configured() bool {
	return c.cfg.ClientID != "" && c.cfg.RedirectURI != ""
}

// SignIn exchanges the authorization code (with the server-held PKCE verifier
// and the device_id VK issued alongside the code) and returns the verified
// identity. state must be the original value generated for this attempt — VK
// echoes it in the token response and we require the echo to match.
func (c *VKClient) SignIn(ctx context.Context, code, codeVerifier, deviceID, state string) (ExternalIdentity, error) {
	if !c.Configured() {
		return ExternalIdentity{}, fmt.Errorf("auth.VK: not configured")
	}
	tok, err := c.exchange(ctx, code, codeVerifier, deviceID, state)
	if err != nil {
		return ExternalIdentity{}, fmt.Errorf("auth.VK: exchange: %w", err)
	}
	id, err := c.fetchIdentity(ctx, tok.AccessToken, tok.UserID.String())
	if err != nil {
		return ExternalIdentity{}, fmt.Errorf("auth.VK: identity: %w", err)
	}
	return id, nil
}

type vkTokenResponse struct {
	AccessToken string      `json:"access_token"`
	UserID      json.Number `json:"user_id"`
	State       string      `json:"state"`
}

func (c *VKClient) exchange(ctx context.Context, code, codeVerifier, deviceID, state string) (vkTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	form.Set("client_id", c.cfg.ClientID)
	form.Set("redirect_uri", c.cfg.RedirectURI)
	form.Set("device_id", deviceID)
	form.Set("state", state)

	body, err := c.postForm(ctx, "/oauth2/auth", form)
	if err != nil {
		return vkTokenResponse{}, err
	}

	var tok vkTokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return vkTokenResponse{}, fmt.Errorf("decode token response: %w", err)
	}
	if tok.AccessToken == "" || tok.UserID.String() == "" {
		// VK returns errors with HTTP 200 in some paths — treat a payload
		// without tokens as a rejection, not a transport fault.
		return vkTokenResponse{}, fmt.Errorf("%w: %s", ErrOAuthRejected, oauthErrorCode(body))
	}
	if tok.State != state {
		return vkTokenResponse{}, fmt.Errorf("%w: state echo mismatch", ErrOAuthRejected)
	}
	return tok, nil
}

// fetchIdentity reads the profile via /oauth2/user_info (which validates the
// access token against our client_id) and cross-checks the user_id against
// the one returned by the code exchange.
func (c *VKClient) fetchIdentity(ctx context.Context, accessToken, expectedUserID string) (ExternalIdentity, error) {
	form := url.Values{}
	form.Set("access_token", accessToken)
	form.Set("client_id", c.cfg.ClientID)

	body, err := c.postForm(ctx, "/oauth2/user_info", form)
	if err != nil {
		return ExternalIdentity{}, err
	}

	var info struct {
		User struct {
			UserID json.Number `json:"user_id"`
			Email  string      `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return ExternalIdentity{}, fmt.Errorf("decode user_info: %w", err)
	}
	sub := info.User.UserID.String()
	if sub == "" {
		return ExternalIdentity{}, fmt.Errorf("%w: %s", ErrOAuthRejected, oauthErrorCode(body))
	}
	if sub != expectedUserID {
		return ExternalIdentity{}, fmt.Errorf("%w: user_id mismatch", ErrOAuthRejected)
	}
	return ExternalIdentity{
		Provider: ProviderVK,
		Subject:  sub,
		Email:    info.User.Email,
		// VK ID does not document an email-confirmation guarantee — never
		// treat the address as verified (no auto-linking by it).
		EmailVerified: false,
	}, nil
}

func (c *VKClient) postForm(ctx context.Context, path string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.BaseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", path, err)
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, fmt.Errorf("%w: %s status %d (%s)", ErrOAuthRejected, path, resp.StatusCode, oauthErrorCode(body))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s status %d", path, resp.StatusCode)
	}
	return body, nil
}
