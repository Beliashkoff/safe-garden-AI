package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/ctxkey"
)

type stubParser struct {
	id  uuid.UUID
	err error
}

func (s stubParser) Parse(string) (authpkg.Claims, error) {
	if s.err != nil {
		return authpkg.Claims{}, s.err
	}
	return authpkg.Claims{Sub: s.id}, nil
}

func TestRequireAuth_PassesAndInjectsUserID(t *testing.T) {
	want := uuid.New()
	var gotID uuid.UUID
	var gotOK bool

	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotID, gotOK = ctxkey.UserID(r.Context())
	})
	h := RequireAuth(stubParser{id: want})(next)

	req := httptest.NewRequest(http.MethodGet, "/account", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.True(t, gotOK)
	assert.Equal(t, want, gotID)
}

func TestRequireAuth_RejectsMissingHeader(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	h := RequireAuth(stubParser{id: uuid.New()})(next)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/account", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called, "next must not run")
}

func TestRequireAuth_RejectsMalformedHeader(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	h := RequireAuth(stubParser{id: uuid.New()})(next)

	req := httptest.NewRequest(http.MethodGet, "/account", nil)
	req.Header.Set("Authorization", "token-without-bearer-prefix")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAuth_RejectsInvalidToken(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	h := RequireAuth(stubParser{err: errors.New("bad token")})(next)

	req := httptest.NewRequest(http.MethodGet, "/account", nil)
	req.Header.Set("Authorization", "Bearer whatever")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
