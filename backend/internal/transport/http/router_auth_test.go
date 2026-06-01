package httptransport_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	httptransport "github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/handler"
)

// errParser rejects every token, so any request reaching RequireAuth — with or
// without a token — is denied. This lets the audit assert 401 on every private
// route without standing up real auth.
type errParser struct{}

func (errParser) Parse(string) (authpkg.Claims, error) {
	return authpkg.Claims{}, errors.New("invalid")
}

// publicRoutes are the ONLY endpoints allowed to be reachable without auth
// (ARCH §4, CLAUDE.md invariant #2). Adding an entry here is a deliberate
// security decision; the test below fails for any other unauthenticated route.
var publicRoutes = map[string]bool{
	"POST /auth/apple":         true,
	"POST /auth/google":        true,
	"POST /auth/email/request": true,
	"POST /auth/email/verify":  true,
	"POST /auth/refresh":       true,
	"POST /auth/logout":        true,
}

// TestAllPrivateRoutesRequireAuth walks every registered /v1 route and asserts
// that anything not in publicRoutes rejects both missing and invalid tokens with
// 401. It turns invariant #2 into a regression guard: a new private endpoint
// added without RequireAuth fails this test.
func TestAllPrivateRoutesRequireAuth(t *testing.T) {
	r := httptransport.NewRouter(httptransport.Deps{
		Handler:     handler.New(nil, nil, nil),
		TokenParser: errParser{},
		DocsEnabled: false,
	})

	var walked, checked int
	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		walked++
		if publicRoutes[method+" "+route] {
			return nil
		}
		checked++
		path := replaceRouteParams(route)

		// Missing Authorization header.
		recNoAuth := httptest.NewRecorder()
		r.ServeHTTP(recNoAuth, httptest.NewRequest(method, path, nil))
		assert.Equalf(t, http.StatusUnauthorized, recNoAuth.Code,
			"private route %s %s must reject requests with no token", method, route)

		// Present but invalid token (must be 401, not 200/500).
		reqBad := httptest.NewRequest(method, path, nil)
		reqBad.Header.Set("Authorization", "Bearer invalid")
		recBad := httptest.NewRecorder()
		r.ServeHTTP(recBad, reqBad)
		assert.Equalf(t, http.StatusUnauthorized, recBad.Code,
			"private route %s %s must reject invalid tokens", method, route)
		return nil
	})
	require.NoError(t, err)
	require.Positive(t, walked, "router exposed no routes — wiring broken")
	require.Positive(t, checked, "no private routes were audited")
}

// replaceRouteParams swaps chi {param} path segments for a concrete value so the
// generated path matches the route pattern during dispatch.
func replaceRouteParams(route string) string {
	parts := strings.Split(route, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			parts[i] = "00000000-0000-0000-0000-000000000000"
		}
	}
	return strings.Join(parts, "/")
}
