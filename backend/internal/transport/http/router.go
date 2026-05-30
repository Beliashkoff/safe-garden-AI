// Package httptransport assembles the versioned (/v1) API router from the
// handler set, the auth middleware, and the optional docs endpoints. It is
// mounted under /v1 by cmd/api; health probes stay at the root.
package httptransport

import (
	"github.com/go-chi/chi/v5"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/docs"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/handler"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/middleware"
)

// Deps are the router's collaborators.
type Deps struct {
	Handler     *handler.Handler
	TokenParser middleware.AccessTokenParser
	DocsEnabled bool
}

// NewRouter builds the /v1 sub-router. Public /auth routes need no token; the
// /account routes sit behind RequireAuth.
func NewRouter(d Deps) chi.Router {
	r := chi.NewRouter()

	r.Route("/auth", func(r chi.Router) {
		r.Post("/apple", d.Handler.SignInApple)
		r.Post("/google", d.Handler.SignInGoogle)
		r.Post("/email/request", d.Handler.RequestOTP)
		r.Post("/email/verify", d.Handler.VerifyOTP)
		r.Post("/refresh", d.Handler.Refresh)
		r.Post("/logout", d.Handler.Logout)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(d.TokenParser))
		r.Get("/account", d.Handler.GetAccount)
		r.Delete("/account", d.Handler.DeleteAccount)

		// Chat (stage 2.3).
		r.Post("/messages", d.Handler.PostMessage)
		r.Delete("/messages/{id}", d.Handler.DeleteMessage)
		r.Get("/conversation", d.Handler.GetConversation)
		r.Get("/conversation/messages", d.Handler.ListMessages)

		// Uploads (stage 3.1) — presigned photo upload.
		r.Post("/uploads/presign", d.Handler.PostPresign)
	})

	if d.DocsEnabled {
		docs.Mount(r)
	}

	return r
}
