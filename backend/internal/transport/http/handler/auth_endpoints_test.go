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

// startOAuth runs POST /v1/auth/{provider}/start and returns the response.
func (h *harness) startOAuth(t *testing.T, provider string) oauthStartResp {
	t.Helper()
	resp, data := h.postJSON(t, "/v1/auth/"+provider+"/start", nil)
	require.Equalf(t, http.StatusOK, resp.StatusCode, "start body: %s", data)
	var out oauthStartResp
	require.NoError(t, json.Unmarshal(data, &out))
	require.NotEmpty(t, out.State)
	return out
}

type oauthStartResp struct {
	State         string `json:"state"`
	CodeChallenge string `json:"code_challenge"`
	AuthURL       string `json:"auth_url"`
}

func (h *harness) signInYandex(t *testing.T, sub, email string) signInResp {
	t.Helper()
	start := h.startOAuth(t, "yandex")
	code := h.yandex.addCode(sub, email)
	resp, data := h.postJSON(t, "/v1/auth/yandex/complete",
		map[string]string{"code": code, "state": start.State})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "complete body: %s", data)
	var out signInResp
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

func (h *harness) signInVK(t *testing.T, sub, email string) signInResp {
	t.Helper()
	start := h.startOAuth(t, "vk")
	code, deviceID := h.vk.addCode(sub, email)
	resp, data := h.postJSON(t, "/v1/auth/vk/complete",
		map[string]string{"code": code, "state": start.State, "device_id": deviceID})
	require.Equalf(t, http.StatusOK, resp.StatusCode, "complete body: %s", data)
	var out signInResp
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

func TestYandex_StartReturnsAuthURL(t *testing.T) {
	h := newHarness(t)
	start := h.startOAuth(t, "yandex")
	assert.Contains(t, start.AuthURL, "/authorize?")
	assert.Contains(t, start.AuthURL, "code_challenge_method=S256")
	assert.Contains(t, start.AuthURL, "client_id="+testYandexClientID)
	assert.Empty(t, start.CodeChallenge, "yandex challenge is embedded in auth_url only")
}

func TestYandex_SignInCreatesAndReuses(t *testing.T) {
	h := newHarness(t)
	first := h.signInYandex(t, "100500", "ya-user@yandex.ru")
	assert.True(t, first.User.Providers.Yandex)
	assert.Equal(t, "ya-user@yandex.ru", first.User.Email)
	assert.True(t, first.User.EmailVerified)

	// Second sign-in with the same subject returns the same account.
	second := h.signInYandex(t, "100500", "ya-user@yandex.ru")
	assert.Equal(t, first.User.ID, second.User.ID)
}

func TestYandex_InvalidState(t *testing.T) {
	h := newHarness(t)
	code := h.yandex.addCode("100501", "x@yandex.ru")
	resp, _ := h.postJSON(t, "/v1/auth/yandex/complete",
		map[string]string{"code": code, "state": "forged-state-value-aaaaaaaaaaaaaa"})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestYandex_StateIsSingleUse(t *testing.T) {
	h := newHarness(t)
	start := h.startOAuth(t, "yandex")
	code := h.yandex.addCode("100502", "y@yandex.ru")
	resp, _ := h.postJSON(t, "/v1/auth/yandex/complete",
		map[string]string{"code": code, "state": start.State})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Replaying the consumed state fails even with a fresh provider code.
	code2 := h.yandex.addCode("100502", "y@yandex.ru")
	resp, _ = h.postJSON(t, "/v1/auth/yandex/complete",
		map[string]string{"code": code2, "state": start.State})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestYandex_StateProviderMismatch(t *testing.T) {
	h := newHarness(t)
	// A state issued for VK must not complete a Yandex sign-in.
	start := h.startOAuth(t, "vk")
	code := h.yandex.addCode("100503", "z@yandex.ru")
	resp, _ := h.postJSON(t, "/v1/auth/yandex/complete",
		map[string]string{"code": code, "state": start.State})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestVK_StartReturnsChallenge(t *testing.T) {
	h := newHarness(t)
	start := h.startOAuth(t, "vk")
	assert.NotEmpty(t, start.CodeChallenge)
	assert.Empty(t, start.AuthURL, "vk UI is built by the native SDK")
}

func TestVK_SignIn(t *testing.T) {
	h := newHarness(t)
	res := h.signInVK(t, "777001", "vk-user@example.com")
	assert.True(t, res.User.Providers.VK)
	// VK email is not provider-verified → not stored on the account.
	assert.Empty(t, res.User.Email)
	assert.False(t, res.User.EmailVerified)
}

func TestVK_WrongDeviceID(t *testing.T) {
	h := newHarness(t)
	start := h.startOAuth(t, "vk")
	code, _ := h.vk.addCode("777002", "")
	resp, _ := h.postJSON(t, "/v1/auth/vk/complete",
		map[string]string{"code": code, "state": start.State, "device_id": "wrong-device"})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAutoLinkByEmail_Yandex(t *testing.T) {
	h := newHarness(t)
	emailUser := h.signInEmail(t, "link@yandex.ru")

	linked := h.signInYandex(t, "100600", "link@yandex.ru")
	assert.Equal(t, emailUser.User.ID, linked.User.ID, "yandex sign-in should attach to the email account")
	assert.True(t, linked.User.Providers.Yandex)
	assert.True(t, linked.User.Providers.Email)
}

func TestVK_NeverAutoLinksByEmail(t *testing.T) {
	h := newHarness(t)
	emailUser := h.signInEmail(t, "owner@example.com")

	// VK reports the same email, but it is not provider-verified — a separate
	// account is created instead of attaching to the OTP one (anti-takeover).
	vkUser := h.signInVK(t, "777003", "owner@example.com")
	assert.NotEqual(t, emailUser.User.ID, vkUser.User.ID)
	assert.Empty(t, vkUser.User.Email)
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

func TestValidation_OAuthCompleteEmptyBody(t *testing.T) {
	h := newHarness(t)
	for _, path := range []string{"/v1/auth/yandex/complete", "/v1/auth/vk/complete"} {
		resp, data := h.postJSON(t, path, map[string]string{})
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode, "%s body: %s", path, data)
		var e errorResp
		require.NoError(t, json.Unmarshal(data, &e))
		assert.Equal(t, "validation_failed", e.Error.Code)
		assert.NotEmpty(t, e.RequestID)
	}
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
