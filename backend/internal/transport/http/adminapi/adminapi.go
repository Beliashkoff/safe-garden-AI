// Package adminapi is the HTTP surface of the admin panel (/admin/v1). It is
// mounted on the public listener but reached through the admin.<domain> Caddy
// site; auth is a cookie-bound opaque session (no JWT — the panel is a
// browser, not the mobile app). Handlers are thin per the project layering.
package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
	adminuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/admin"
)

// maxBodyBytes caps admin JSON bodies (the longest field is long_desc, 4000
// chars). Image upload uses its own multipart cap.
const maxBodyBytes = 64 * 1024

// sessionCookie carries the opaque admin session token. Path-restricted to
// /admin so the browser never attaches it to the public API routes.
const sessionCookie = "sg_admin"

// Handler serves the admin endpoints.
type Handler struct {
	svc           *adminuc.Service
	cookieSecure  bool
	allowedOrigin string
	logger        *slog.Logger
}

// New constructs the handler set. cookieSecure must be true in prod (TLS via
// Caddy); allowedOrigin guards mutating requests against cross-site calls.
func New(svc *adminuc.Service, cookieSecure bool, allowedOrigin string, logger *slog.Logger) *Handler {
	return &Handler{
		svc:           svc,
		cookieSecure:  cookieSecure,
		allowedOrigin: allowedOrigin,
		logger:        logger,
	}
}

// Routes builds the /admin/v1 router.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(h.checkOrigin)

	r.Route("/auth", func(r chi.Router) {
		r.Get("/status", h.getStatus)
		r.Post("/setup/request", h.postSetupRequest)
		r.Post("/setup/complete", h.postSetupComplete)
		r.Post("/login", h.postLogin)
		r.Post("/logout", h.postLogout)
		r.Post("/reset/request", h.postResetRequest)
		r.Post("/reset/complete", h.postResetComplete)
	})

	r.Group(func(r chi.Router) {
		r.Use(h.requireAdmin)
		r.Get("/me", h.getMe)
		r.Post("/me/password", h.postChangePassword)
		r.Get("/sessions", h.getSessions)
		r.Delete("/sessions/{id}", h.deleteSession)

		r.Get("/fertilizers", h.getProducts)
		r.Post("/fertilizers", h.postProduct)
		r.Get("/fertilizers/problems", h.getProblemKeys)
		r.Post("/fertilizers/image", h.postProductImage)
		r.Get("/fertilizers/{id}", h.getProduct)
		r.Put("/fertilizers/{id}", h.putProduct)
		r.Delete("/fertilizers/{id}", h.deleteProduct)

		r.Get("/stats/overview", h.getOverview)
		r.Get("/stats/timeseries", h.getTimeseries)
		r.Get("/stats/top-products", h.getTopProducts)
		r.Get("/stats/feedback", h.getFeedback)
		r.Get("/stats/message-status", h.getMessageStatus)
		r.Get("/stats/login-breakdown", h.getLoginBreakdown)
		r.Get("/stats/top-users", h.getTopUsers)
		r.Get("/stats/errors-breakdown", h.getErrorsBreakdown)

		// Growth analytics ("Рост").
		r.Get("/stats/activation", h.getActivation)
		r.Get("/stats/retention", h.getRetention)
		r.Get("/stats/conversation-depth", h.getConversationDepth)
		r.Get("/stats/heatmap", h.getHeatmap)
		r.Get("/stats/activity-weekly", h.getActivityWeekly)

		// Answer quality ("Качество ответов").
		r.Get("/stats/ctr", h.getCTR)
		r.Get("/stats/length-vs-verdict", h.getLengthVsVerdict)
		r.Get("/stats/negative-conversations", h.getNegativeConversations)
		r.Get("/stats/followup", h.getFollowup)
		r.Get("/stats/downvoted", h.getDownvoted)

		// Cost / FinOps ("Стоимость").
		r.Get("/stats/unit-economics", h.getUnitEconomics)
		r.Get("/stats/cache", h.getCacheStats)
		r.Get("/stats/cost-by-model", h.getCostByModel)
		r.Get("/stats/cost-by-kind", h.getCostByKind)

		// Reliability & ops ("Надёжность").
		r.Get("/health/worker", h.getWorkerHealth)
		r.Get("/health/deps", h.getDependencies)
		r.Get("/stats/fail-codes", h.getFailCodes)
		r.Get("/alerts", h.getAlerts)

		// Security & abuse ("Безопасность").
		r.Get("/security/events", h.getSecurityEvents)
		r.Get("/security/otp", h.getOtpStats)
		r.Get("/security/suspicious-ips", h.getSuspiciousIPs)

		// Data lifecycle & compliance ("Данные и комплаенс").
		r.Get("/compliance/deletions", h.getDeletionEvents)
		r.Get("/compliance/stale-purges", h.getStalePurges)
		r.Get("/compliance/overview", h.getComplianceOverview)

		r.Get("/errors", h.getErrors)
		r.Get("/audit", h.getAudit)
	})

	return r
}

// --- middleware ---

type ctxKey int

const (
	ctxAdminID ctxKey = iota
	ctxSessionID
)

// checkOrigin rejects mutating cross-origin requests. SameSite=Strict on the
// cookie already blocks classic CSRF; this is defense-in-depth for older
// browsers. Requests without an Origin header (curl, same-origin GET) pass.
func (h *Handler) checkOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.allowedOrigin != "" && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			if origin := r.Header.Get("Origin"); origin != "" && origin != h.allowedOrigin {
				httperr.Write(w, r, httperr.Forbidden("cross-origin request rejected"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// requireAdmin resolves the session cookie, stores admin+session ids in the
// context, and rejects with 401 otherwise.
func (h *Handler) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil {
			httperr.Write(w, r, httperr.Unauthorized("authentication required"))
			return
		}
		sess, err := h.svc.ValidateSession(r.Context(), c.Value)
		if err != nil {
			if errors.Is(err, adminuc.ErrSessionInvalid) {
				h.clearSessionCookie(w)
				httperr.Write(w, r, httperr.Unauthorized("session expired, sign in again"))
				return
			}
			respondError(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), ctxAdminID, sess.AdminID)
		ctx = context.WithValue(ctx, ctxSessionID, sess.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func adminIDFrom(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(ctxAdminID).(uuid.UUID)
	return id
}

func sessionIDFrom(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(ctxSessionID).(uuid.UUID)
	return id
}

// --- cookies ---

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: HttpOnly + SameSiteStrict set below; Secure is env-driven (true in prod, false for local http dev)
		Name:     sessionCookie,
		Value:    token,
		Path:     "/admin",
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: HttpOnly + SameSiteStrict set below; Secure is env-driven (true in prod, false for local http dev)
		Name:     sessionCookie,
		Value:    "",
		Path:     "/admin",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

// --- shared helpers (mirror handler package idioms) ---

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return httperr.PayloadTooLarge("request body too large")
		}
		return httperr.ValidationFailed("invalid JSON body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode admin response", "err", err.Error())
	}
}

func respondError(w http.ResponseWriter, r *http.Request, err error) {
	httperr.Write(w, r, mapErr(err))
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, adminuc.ErrAlreadyBootstrapped):
		return httperr.Forbidden("admin account already exists")
	case errors.Is(err, adminuc.ErrNotBootstrapped):
		return httperr.NotFound("admin account is not set up yet")
	case errors.Is(err, adminuc.ErrEmailNotAllowed):
		return httperr.Forbidden("this email cannot manage the admin panel")
	case errors.Is(err, adminuc.ErrInvalidCredentials):
		return httperr.Unauthorized("invalid email or password")
	case errors.Is(err, adminuc.ErrInvalidCode):
		return httperr.Unauthorized("invalid or expired code")
	case errors.Is(err, adminuc.ErrTooManyAttempts):
		return httperr.RateLimited("too many attempts for this code")
	case errors.Is(err, adminuc.ErrRateLimited):
		return httperr.RateLimited("too many requests, try again later")
	case errors.Is(err, adminuc.ErrWeakPassword):
		return httperr.ValidationFailed("password must be at least 12 characters with letters and digits").WithDetail("field", "password")
	case errors.Is(err, adminuc.ErrSessionInvalid):
		return httperr.Unauthorized("session expired, sign in again")
	case errors.Is(err, adminuc.ErrProductNotFound):
		return httperr.NotFound("product not found")
	case errors.Is(err, adminuc.ErrSlugTaken):
		return httperr.ValidationFailed("slug already exists").WithDetail("field", "slug")
	case errors.Is(err, adminuc.ErrValidation):
		return httperr.ValidationFailed(err.Error())
	default:
		return err
	}
}

func deviceMetaFrom(r *http.Request) adminuc.DeviceMeta {
	return adminuc.DeviceMeta{
		UserAgent: r.UserAgent(),
		IP:        clientIP(r),
	}
}

func clientIP(r *http.Request) netip.Addr {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return addr
}
