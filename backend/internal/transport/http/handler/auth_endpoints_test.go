//go:build integration

package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) signInEmail(t *testing.T, email string) signInResp {
	t.Helper()
	resp, _ := h.postJSON(t, "/v1/auth/email/request", map[string]string{"email": email})
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	code := h.mailer.code()
	require.NotEmpty(t, code)

	resp, data := h.postJSON(t, "/v1/auth/email/verify", map[string]string{"email": email, "code": code})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "verify body: %s", data)
	var out signInResp
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

func TestEmailOTP_FullFlow(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "alice@example.com")

	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)
	assert.Equal(t, "alice@example.com", res.User.Email)
	assert.True(t, res.User.EmailVerified)
	assert.True(t, res.User.Providers.Email)

	resp, data := h.do(t, http.MethodGet, "/v1/account", nil, bearer(res.AccessToken))
	require.Equalf(t, http.StatusOK, resp.StatusCode, "account body: %s", data)
	var acc struct {
		User struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	require.NoError(t, json.Unmarshal(data, &acc))
	assert.Equal(t, res.User.ID, acc.User.ID)
	assert.Equal(t, "alice@example.com", acc.User.Email)
}

func TestEmailVerify_WrongCodeThenCap(t *testing.T) {
	h := newHarness(t)
	resp, _ := h.postJSON(t, "/v1/auth/email/request", map[string]string{"email": "bob@example.com"})
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// 5 wrong attempts → 401 each.
	for i := 0; i < 5; i++ {
		resp, _ := h.postJSON(t, "/v1/auth/email/verify", map[string]string{"email": "bob@example.com", "code": "000000"})
		require.Equalf(t, http.StatusUnauthorized, resp.StatusCode, "attempt %d", i+1)
	}
	// 6th attempt is over the cap → 429, even with the right code.
	resp, _ = h.postJSON(t, "/v1/auth/email/verify", map[string]string{"email": "bob@example.com", "code": h.mailer.code()})
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

func TestEmailVerify_BadFormat(t *testing.T) {
	h := newHarness(t)
	resp, _ := h.postJSON(t, "/v1/auth/email/verify", map[string]string{"email": "x@example.com", "code": "12"})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestEmailRequest_RateLimited(t *testing.T) {
	h := newHarness(t)
	for i := 0; i < 3; i++ {
		resp, _ := h.postJSON(t, "/v1/auth/email/request", map[string]string{"email": "rl@example.com"})
		require.Equalf(t, http.StatusNoContent, resp.StatusCode, "request %d", i+1)
	}
	resp, data := h.postJSON(t, "/v1/auth/email/request", map[string]string{"email": "rl@example.com"})
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	var e errorResp
	require.NoError(t, json.Unmarshal(data, &e))
	assert.Equal(t, "rate_limited", e.Error.Code)
}

func TestRefresh_RotationAndReuseDetection(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "carol@example.com")

	resp, data := h.postJSON(t, "/v1/auth/refresh", map[string]string{"refresh_token": res.RefreshToken})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "refresh body: %s", data)
	var rotated struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(data, &rotated))
	assert.NotEmpty(t, rotated.RefreshToken)
	assert.NotEqual(t, res.RefreshToken, rotated.RefreshToken)

	// Reusing the old (now revoked) token is treated as theft → 401.
	resp, _ = h.postJSON(t, "/v1/auth/refresh", map[string]string{"refresh_token": res.RefreshToken})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// And the whole family is revoked, so the rotated token no longer works.
	resp, _ = h.postJSON(t, "/v1/auth/refresh", map[string]string{"refresh_token": rotated.RefreshToken})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestLogout_RevokesRefresh(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "dave@example.com")

	resp, _ := h.postJSON(t, "/v1/auth/logout", map[string]string{"refresh_token": res.RefreshToken})
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp, _ = h.postJSON(t, "/v1/auth/refresh", map[string]string{"refresh_token": res.RefreshToken})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestApple_SignInCreatesAndReuses(t *testing.T) {
	h := newHarness(t)
	const nonce = "nonce-apple-1"
	tok := h.idp.appleToken(t, "apple-sub-1", "apple-user@example.com", nonce, true)

	resp, data := h.postJSON(t, "/v1/auth/apple", map[string]string{"id_token": tok, "nonce": nonce})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "apple body: %s", data)
	var first signInResp
	require.NoError(t, json.Unmarshal(data, &first))
	assert.True(t, first.User.Providers.Apple)

	// Second sign-in with the same subject returns the same account.
	tok2 := h.idp.appleToken(t, "apple-sub-1", "apple-user@example.com", nonce, true)
	_, data = h.postJSON(t, "/v1/auth/apple", map[string]string{"id_token": tok2, "nonce": nonce})
	var second signInResp
	require.NoError(t, json.Unmarshal(data, &second))
	assert.Equal(t, first.User.ID, second.User.ID)
}

func TestApple_NonceMismatch(t *testing.T) {
	h := newHarness(t)
	tok := h.idp.appleToken(t, "apple-sub-2", "x@example.com", "real-nonce", true)
	resp, _ := h.postJSON(t, "/v1/auth/apple", map[string]string{"id_token": tok, "nonce": "wrong-nonce"})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGoogle_SignIn(t *testing.T) {
	h := newHarness(t)
	tok := h.idp.googleToken(t, "google-sub-1", "g-user@example.com", true)
	resp, data := h.postJSON(t, "/v1/auth/google", map[string]string{"id_token": tok})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "google body: %s", data)
	var res signInResp
	require.NoError(t, json.Unmarshal(data, &res))
	assert.True(t, res.User.Providers.Google)
}

func TestAutoLinkByEmail(t *testing.T) {
	h := newHarness(t)
	emailUser := h.signInEmail(t, "link@example.com")

	tok := h.idp.googleToken(t, "google-sub-link", "link@example.com", true)
	resp, data := h.postJSON(t, "/v1/auth/google", map[string]string{"id_token": tok})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "google body: %s", data)
	var linked signInResp
	require.NoError(t, json.Unmarshal(data, &linked))

	assert.Equal(t, emailUser.User.ID, linked.User.ID, "google sign-in should attach to the email account")
	assert.True(t, linked.User.Providers.Google)
	assert.True(t, linked.User.Providers.Email)
}

func TestRequireAuth_RejectsMissingAndBadToken(t *testing.T) {
	h := newHarness(t)

	resp, _ := h.do(t, http.MethodGet, "/v1/account", nil, nil)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	resp, _ = h.do(t, http.MethodGet, "/v1/account", nil, bearer("not-a-jwt"))
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestDeleteAccount(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "erin@example.com")

	resp, _ := h.do(t, http.MethodDelete, "/v1/account", nil, bearer(res.AccessToken))
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Token still cryptographically valid, but the user row is gone → 404.
	resp, _ = h.do(t, http.MethodGet, "/v1/account", nil, bearer(res.AccessToken))
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Refresh tokens were revoked by deletion.
	resp, _ = h.postJSON(t, "/v1/auth/refresh", map[string]string{"refresh_token": res.RefreshToken})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestValidation_AppleEmptyBody(t *testing.T) {
	h := newHarness(t)
	resp, data := h.postJSON(t, "/v1/auth/apple", map[string]string{})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	var e errorResp
	require.NoError(t, json.Unmarshal(data, &e))
	assert.Equal(t, "validation_failed", e.Error.Code)
	assert.NotEmpty(t, e.RequestID)
}

func TestDocs_Served(t *testing.T) {
	h := newHarness(t)

	resp, _ := h.do(t, http.MethodGet, "/v1/docs", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	resp, data := h.do(t, http.MethodGet, "/v1/docs/openapi.yaml", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(data), "openapi: 3.0")
}
