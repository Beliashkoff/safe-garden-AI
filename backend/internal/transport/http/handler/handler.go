// Package handler contains the HTTP handlers for the auth + account endpoints.
// Handlers are thin: decode/validate input, call the usecase, map domain errors
// to the §4.7 envelope, and serialize the result. No business logic lives here.
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
	authuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/auth"
	chatuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/chat"
)

// maxAuthBodyBytes caps auth request bodies (ARCH §8.2 text body ≤ 32 KB).
const maxAuthBodyBytes = 32 * 1024

// Handler serves the auth + account + chat endpoints.
type Handler struct {
	svc  *authuc.Service
	chat *chatuc.Service
}

// New constructs the handler set.
func New(svc *authuc.Service, chat *chatuc.Service) *Handler {
	return &Handler{svc: svc, chat: chat}
}

// decodeJSON reads a size-limited JSON body, rejecting unknown fields. It
// returns a ready-to-write *httperr.Error on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)
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
		slog.ErrorContext(r.Context(), "failed to encode response", "err", err.Error())
	}
}

// respondError maps a usecase sentinel error to its HTTP shape. Unknown errors
// fall through to httperr.Write, which logs them and returns a generic 500.
func respondError(w http.ResponseWriter, r *http.Request, err error) {
	httperr.Write(w, r, mapErr(err))
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, authuc.ErrInvalidIDToken):
		return httperr.Unauthorized("invalid id_token")
	case errors.Is(err, authuc.ErrInvalidEmail):
		return httperr.ValidationFailed("invalid email").WithDetail("field", "email")
	case errors.Is(err, authuc.ErrInvalidOTP):
		return httperr.Unauthorized("invalid or expired code")
	case errors.Is(err, authuc.ErrTooManyAttempts):
		return httperr.RateLimited("too many attempts for this code")
	case errors.Is(err, authuc.ErrRateLimited):
		return httperr.RateLimited("too many requests, try again later")
	case errors.Is(err, authuc.ErrInvalidToken):
		return httperr.Unauthorized("invalid refresh token")
	case errors.Is(err, authuc.ErrUserNotFound):
		return httperr.NotFound("account not found")
	case errors.Is(err, chatuc.ErrEmptyContent):
		return httperr.ValidationFailed("message content is empty")
	case errors.Is(err, chatuc.ErrUnsupportedBlock):
		return httperr.UnsupportedMedia("only text content is supported")
	case errors.Is(err, chatuc.ErrTextTooLarge):
		return httperr.PayloadTooLarge("message text is too large")
	case errors.Is(err, chatuc.ErrBadCursor):
		return httperr.ValidationFailed("invalid cursor")
	case errors.Is(err, chatuc.ErrRateLimited):
		return httperr.RateLimited("too many messages, slow down")
	case errors.Is(err, chatuc.ErrMessageNotFound):
		return httperr.NotFound("message not found")
	default:
		return err
	}
}

// deviceMetaFrom collects optional session/audit context from the request.
func deviceMetaFrom(r *http.Request) authuc.DeviceMeta {
	return authuc.DeviceMeta{
		UserAgent: r.UserAgent(),
		DeviceID:  r.Header.Get("X-Device-ID"),
		IP:        clientIP(r),
	}
}

// clientIP parses r.RemoteAddr (set to the real client IP by chi's RealIP
// middleware). Returns the zero Addr when unparseable.
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

func localeFrom(r *http.Request) string {
	if al := r.Header.Get("Accept-Language"); len(al) >= 2 {
		return al[:2]
	}
	return "ru"
}

// --- response DTOs (mirror ARCH §4.1) ---

type userDTO struct {
	ID            string       `json:"id"`
	Email         string       `json:"email,omitempty"`
	DisplayName   string       `json:"display_name,omitempty"`
	EmailVerified bool         `json:"email_verified"`
	Providers     providersDTO `json:"providers"`
}

type providersDTO struct {
	Apple  bool `json:"apple"`
	Google bool `json:"google"`
	Email  bool `json:"email"`
}

func toUserDTO(u authuc.UserView) userDTO {
	return userDTO{
		ID:            u.ID.String(),
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		EmailVerified: u.EmailVerified,
		Providers: providersDTO{
			Apple:  u.HasApple,
			Google: u.HasGoogle,
			Email:  u.Email != "",
		},
	}
}

type signInResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	User         userDTO `json:"user"`
}

func toSignInResponse(res authuc.AuthResult) signInResponse {
	return signInResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		User:         toUserDTO(res.User),
	}
}
