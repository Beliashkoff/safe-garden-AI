package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	vkTestClientID = "53000000"
	vkTestRedirect = "vk53000000://vk.ru"
)

type vkFixture struct {
	authStatus int
	authBody   func(form url.Values) string
	infoStatus int
	infoBody   string

	lastAuthForm url.Values
	lastInfoForm url.Values
}

func newVKFixture(t *testing.T) (*vkFixture, *VKClient) {
	t.Helper()
	f := &vkFixture{
		authStatus: http.StatusOK,
		authBody: func(form url.Values) string {
			return fmt.Sprintf(`{"access_token":"vk-acc-1","refresh_token":"r","id_token":"i","token_type":"Bearer","expires_in":3600,"user_id":777001,"state":"%s","scope":"vkid.personal_info email"}`,
				form.Get("state"))
		},
		infoStatus: http.StatusOK,
		infoBody:   `{"user":{"user_id":"777001","first_name":"Test","last_name":"User","email":"user@example.com"}}`,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/auth", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		f.lastAuthForm = r.PostForm
		w.WriteHeader(f.authStatus)
		fmt.Fprint(w, f.authBody(r.PostForm))
	})
	mux.HandleFunc("/oauth2/user_info", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		f.lastInfoForm = r.PostForm
		w.WriteHeader(f.infoStatus)
		fmt.Fprint(w, f.infoBody)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewVK(VKConfig{ClientID: vkTestClientID, RedirectURI: vkTestRedirect, BaseURL: srv.URL})
	return f, c
}

func TestVKSignIn_Success(t *testing.T) {
	f, c := newVKFixture(t)

	id, err := c.SignIn(context.Background(), "the-code", "the-verifier", "device-1", "the-state")
	require.NoError(t, err)
	assert.Equal(t, ProviderVK, id.Provider)
	assert.Equal(t, "777001", id.Subject)
	assert.Equal(t, "user@example.com", id.Email)
	assert.False(t, id.EmailVerified, "vk email must never be treated as verified")

	assert.Equal(t, "authorization_code", f.lastAuthForm.Get("grant_type"))
	assert.Equal(t, "the-code", f.lastAuthForm.Get("code"))
	assert.Equal(t, "the-verifier", f.lastAuthForm.Get("code_verifier"))
	assert.Equal(t, "device-1", f.lastAuthForm.Get("device_id"))
	assert.Equal(t, vkTestRedirect, f.lastAuthForm.Get("redirect_uri"))
	assert.Equal(t, vkTestClientID, f.lastInfoForm.Get("client_id"))
	assert.Equal(t, "vk-acc-1", f.lastInfoForm.Get("access_token"))
}

func TestVKSignIn_StateEchoMismatch(t *testing.T) {
	f, c := newVKFixture(t)
	f.authBody = func(url.Values) string {
		return `{"access_token":"a","user_id":777001,"state":"tampered"}`
	}
	_, err := c.SignIn(context.Background(), "c", "v", "d", "expected-state")
	require.ErrorIs(t, err, ErrOAuthRejected)
}

func TestVKSignIn_UserIDMismatch(t *testing.T) {
	f, c := newVKFixture(t)
	// user_info reports a different account than the code exchange.
	f.infoBody = `{"user":{"user_id":"999999","email":"user@example.com"}}`
	_, err := c.SignIn(context.Background(), "c", "v", "d", "s")
	require.ErrorIs(t, err, ErrOAuthRejected)
}

func TestVKSignIn_ErrorWith200(t *testing.T) {
	// VK can return an error payload with HTTP 200 — must be a rejection.
	f, c := newVKFixture(t)
	f.authBody = func(url.Values) string {
		return `{"error":"invalid_request","error_description":"bad code"}`
	}
	_, err := c.SignIn(context.Background(), "c", "v", "d", "s")
	require.ErrorIs(t, err, ErrOAuthRejected)
}

func TestVKSignIn_BadRequest(t *testing.T) {
	f, c := newVKFixture(t)
	f.authStatus = http.StatusBadRequest
	f.authBody = func(url.Values) string { return `{"error":"invalid_request"}` }
	_, err := c.SignIn(context.Background(), "c", "v", "d", "s")
	require.ErrorIs(t, err, ErrOAuthRejected)
}

func TestVKSignIn_ServerError_NotRejected(t *testing.T) {
	f, c := newVKFixture(t)
	f.authStatus = http.StatusServiceUnavailable
	f.authBody = func(url.Values) string { return "unavailable" }
	_, err := c.SignIn(context.Background(), "c", "v", "d", "s")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrOAuthRejected)
}

func TestVKSignIn_NotConfigured(t *testing.T) {
	c := NewVK(VKConfig{})
	assert.False(t, c.Configured())
	_, err := c.SignIn(context.Background(), "c", "v", "d", "s")
	require.Error(t, err)
}
