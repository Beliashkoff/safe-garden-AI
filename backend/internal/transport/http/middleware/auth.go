// Package middleware holds HTTP middleware specific to this service. Generic
// middleware (RequestID, RealIP, Recoverer) come from chi; access logging lives
// in internal/observability.
package middleware

import (
	"net/http"
	"strings"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/ctxkey"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
)

// AccessTokenParser verifies an access token and returns its claims. Satisfied
// by *auth.Issuer; an interface so the middleware is unit-testable.
type AccessTokenParser interface {
	Parse(raw string) (authpkg.Claims, error)
}

const bearerPrefix = "Bearer "

// RequireAuth verifies the Authorization bearer JWT and stores the user_id in
// the request context. Any failure short-circuits with a 401 in the §4.7 shape.
func RequireAuth(parser AccessTokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, bearerPrefix) {
				httperr.Write(w, r, httperr.Unauthorized("missing or malformed Authorization header"))
				return
			}
			claims, err := parser.Parse(strings.TrimPrefix(h, bearerPrefix))
			if err != nil {
				httperr.Write(w, r, httperr.Unauthorized("invalid or expired token"))
				return
			}
			ctx := ctxkey.WithUserID(r.Context(), claims.Sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
