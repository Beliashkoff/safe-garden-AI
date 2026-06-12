package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	yaTestClientID = "ya-client"
	yaTestSecret   = "ya-secret"
	yaTestRedirect = "safegarden://auth/yandex"
)

type yandexFixture struct {
	tokenStatus int
	tokenBody   string
	infoStatus  int
	infoBody    func() string

	lastTokenForm url.Values
	lastAuthz     string
}

func yandexJWT(t *testing.T, secret string, mutate func(jwt.MapClaims)) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss":   "login.yandex.ru",
		"iat":   time.Now().Add(-time.Minute).Unix(),
		"exp":   time.Now().Add(10 * time.Minute).Unix(),
		"uid":   "1000034426",
		"email": "user@yandex.ru",
	}
	if mutate != nil {
		mutate(claims)
	}
	out, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return out
}

func newYandexFixture(t *testing.T) (*yandexFixture, *YandexClient) {
	t.Helper()
	f := &yandexFixture{
		tokenStatus: http.StatusOK,
		tokenBody:   `{"token_type":"bearer","access_token":"acc-1","expires_in":31536000}`,
		infoStatus:  http.StatusOK,
	}
	f.infoBody = func() string { return yandexJWT(t, yaTestSecret, nil) }

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		f.lastTokenForm = r.PostForm
		w.WriteHeader(f.tokenStatus)
		fmt.Fprint(w, f.tokenBody)
	})
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		f.lastAuthz = r.Header.Get("Authorization")
		w.WriteHeader(f.infoStatus)
		fmt.Fprint(w, f.infoBody())
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewYandex(YandexConfig{
		ClientID:     yaTestClientID,
		ClientSecret: yaTestSecret,
		RedirectURI:  yaTestRedirect,
		OAuthBaseURL: srv.URL,
		LoginBaseURL: srv.URL,
	})
	return f, c
}

func TestYandexAuthorizeURL(t *testing.T) {
	c := NewYandex(YandexConfig{ClientID: yaTestClientID, ClientSecret: yaTestSecret, RedirectURI: yaTestRedirect})
	u, err := url.Parse(c.AuthorizeURL("the-state", "the-challenge"))
	require.NoError(t, err)
	assert.Equal(t, "oauth.yandex.ru", u.Host)
	q := u.Query()
	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, yaTestClientID, q.Get("client_id"))
	assert.Equal(t, yaTestRedirect, q.Get("redirect_uri"))
	assert.Equal(t, "login:info login:email", q.Get("scope"))
	assert.Equal(t, "the-state", q.Get("state"))
	assert.Equal(t, "the-challenge", q.Get("code_challenge"))
	assert.Equal(t, "S256", q.Get("code_challenge_method"))
}

func TestYandexSignIn_Success(t *testing.T) {
	f, c := newYandexFixture(t)

	id, err := c.SignIn(context.Background(), "the-code", "the-verifier")
	require.NoError(t, err)
	assert.Equal(t, ProviderYandex, id.Provider)
	assert.Equal(t, "1000034426", id.Subject)
	assert.Equal(t, "user@yandex.ru", id.Email)
	assert.True(t, id.EmailVerified)

	assert.Equal(t, "authorization_code", f.lastTokenForm.Get("grant_type"))
	assert.Equal(t, "the-code", f.lastTokenForm.Get("code"))
	assert.Equal(t, "the-verifier", f.lastTokenForm.Get("code_verifier"))
	assert.Equal(t, yaTestSecret, f.lastTokenForm.Get("client_secret"))
	assert.Equal(t, "OAuth acc-1", f.lastAuthz)
}

func TestYandexSignIn_NumericUIDClaim(t *testing.T) {
	f, c := newYandexFixture(t)
	f.infoBody = func() string {
		return yandexJWT(t, yaTestSecret, func(m jwt.MapClaims) { m["uid"] = 1000034426 })
	}
	id, err := c.SignIn(context.Background(), "c", "v")
	require.NoError(t, err)
	assert.Equal(t, "1000034426", id.Subject)
}

func TestYandexSignIn_BadCode(t *testing.T) {
	f, c := newYandexFixture(t)
	f.tokenStatus = http.StatusBadRequest
	f.tokenBody = `{"error":"invalid_grant","error_description":"Code has expired"}`

	_, err := c.SignIn(context.Background(), "expired", "v")
	require.ErrorIs(t, err, ErrOAuthRejected)
}

func TestYandexSignIn_ServerError_NotRejected(t *testing.T) {
	f, c := newYandexFixture(t)
	f.tokenStatus = http.StatusBadGateway
	f.tokenBody = "bad gateway"

	_, err := c.SignIn(context.Background(), "c", "v")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrOAuthRejected)
}

func TestYandexSignIn_JWTWrongSecret(t *testing.T) {
	// A JWT signed with another app's secret (token substitution) must fail.
	f, c := newYandexFixture(t)
	f.infoBody = func() string { return yandexJWT(t, "other-app-secret", nil) }

	_, err := c.SignIn(context.Background(), "c", "v")
	require.ErrorIs(t, err, ErrOAuthRejected)
}

func TestYandexSignIn_JWTWrongIssuer(t *testing.T) {
	f, c := newYandexFixture(t)
	f.infoBody = func() string {
		return yandexJWT(t, yaTestSecret, func(m jwt.MapClaims) { m["iss"] = "evil.example.com" })
	}
	_, err := c.SignIn(context.Background(), "c", "v")
	require.ErrorIs(t, err, ErrOAuthRejected)
}

func TestYandexSignIn_JWTExpired(t *testing.T) {
	f, c := newYandexFixture(t)
	f.infoBody = func() string {
		return yandexJWT(t, yaTestSecret, func(m jwt.MapClaims) {
			m["exp"] = time.Now().Add(-time.Minute).Unix()
		})
	}
	_, err := c.SignIn(context.Background(), "c", "v")
	require.ErrorIs(t, err, ErrOAuthRejected)
}

func TestYandexSignIn_NotConfigured(t *testing.T) {
	c := NewYandex(YandexConfig{})
	assert.False(t, c.Configured())
	_, err := c.SignIn(context.Background(), "c", "v")
	require.Error(t, err)
}
