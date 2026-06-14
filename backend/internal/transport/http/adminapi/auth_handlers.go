package adminapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
	adminuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/admin"
)

type adminDTO struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	CreatedAt   string `json:"created_at"`
	LastLoginAt string `json:"last_login_at,omitempty"`
}

func toAdminDTO(v adminuc.AdminView) adminDTO {
	dto := adminDTO{
		ID:        v.ID.String(),
		Email:     v.Email,
		CreatedAt: v.CreatedAt.Format(time.RFC3339),
	}
	if !v.LastLoginAt.IsZero() {
		dto.LastLoginAt = v.LastLoginAt.Format(time.RFC3339)
	}
	return dto
}

// getStatus — GET /admin/v1/auth/status. Public: the SPA decides between the
// sign-in screen and the first-time setup wizard.
func (h *Handler) getStatus(w http.ResponseWriter, r *http.Request) {
	ok, err := h.svc.Bootstrapped(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]bool{"bootstrapped": ok})
}

// postSetupRequest — POST /admin/v1/auth/setup/request.
func (h *Handler) postSetupRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	if err := h.svc.RequestSetupCode(r.Context(), req.Email); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// postSetupComplete — POST /admin/v1/auth/setup/complete. Creates the account
// and signs the operator in (cookie set).
func (h *Handler) postSetupComplete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Code     string `json:"code"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	res, err := h.svc.CompleteSetup(r.Context(), req.Email, req.Code, req.Password, deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	h.setSessionCookie(w, res.Token, res.ExpiresAt)
	writeJSON(w, r, http.StatusOK, map[string]any{"admin": toAdminDTO(res.Admin)})
}

// postLogin — POST /admin/v1/auth/login.
func (h *Handler) postLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	res, err := h.svc.Login(r.Context(), req.Email, req.Password, deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	h.setSessionCookie(w, res.Token, res.ExpiresAt)
	writeJSON(w, r, http.StatusOK, map[string]any{"admin": toAdminDTO(res.Admin)})
}

// postLogout — POST /admin/v1/auth/logout. Idempotent.
func (h *Handler) postLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if err := h.svc.Logout(r.Context(), c.Value); err != nil {
			respondError(w, r, err)
			return
		}
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// postResetRequest — POST /admin/v1/auth/reset/request.
func (h *Handler) postResetRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	if err := h.svc.RequestResetCode(r.Context(), req.Email); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// postResetComplete — POST /admin/v1/auth/reset/complete. All sessions are
// revoked; the operator signs in with the new password.
func (h *Handler) postResetComplete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	if err := h.svc.CompleteReset(r.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		respondError(w, r, err)
		return
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// getMe — GET /admin/v1/me.
func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	view, err := h.svc.Me(r.Context(), adminIDFrom(r.Context()))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"admin": toAdminDTO(view)})
}

// postChangePassword — POST /admin/v1/me/password.
func (h *Handler) postChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	err := h.svc.ChangePassword(r.Context(), adminIDFrom(r.Context()), sessionIDFrom(r.Context()),
		req.CurrentPassword, req.NewPassword, deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type sessionDTO struct {
	ID         string `json:"id"`
	IP         string `json:"ip,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	CreatedAt  string `json:"created_at"`
	LastSeenAt string `json:"last_seen_at"`
	Current    bool   `json:"current"`
}

// getSessions — GET /admin/v1/sessions.
func (h *Handler) getSessions(w http.ResponseWriter, r *http.Request) {
	views, err := h.svc.ListSessions(r.Context(), adminIDFrom(r.Context()), sessionIDFrom(r.Context()))
	if err != nil {
		respondError(w, r, err)
		return
	}
	out := make([]sessionDTO, 0, len(views))
	for _, v := range views {
		out = append(out, sessionDTO{
			ID:         v.ID.String(),
			IP:         v.IP,
			UserAgent:  v.UserAgent,
			CreatedAt:  v.CreatedAt.Format(time.RFC3339),
			LastSeenAt: v.LastSeenAt.Format(time.RFC3339),
			Current:    v.Current,
		})
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"sessions": out})
}

// deleteSession — DELETE /admin/v1/sessions/{id}.
func (h *Handler) deleteSession(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httperr.Write(w, r, httperr.ValidationFailed("invalid session id"))
		return
	}
	if err := h.svc.RevokeSession(r.Context(), adminIDFrom(r.Context()), id, deviceMetaFrom(r)); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
