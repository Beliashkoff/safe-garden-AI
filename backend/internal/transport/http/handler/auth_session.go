package handler

import (
	"net/http"
	"strings"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
)

type refreshBody struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Refresh handles POST /v1/auth/refresh — rotates the refresh token.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshBody
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		httperr.Write(w, r, httperr.ValidationFailed("refresh_token is required"))
		return
	}
	res, err := h.svc.Refresh(r.Context(), req.RefreshToken, deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, refreshResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
	})
}

// Logout handles POST /v1/auth/logout — revokes the presented refresh token.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshBody
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		httperr.Write(w, r, httperr.ValidationFailed("refresh_token is required"))
		return
	}
	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
