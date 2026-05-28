package handler

import (
	"net/http"
	"strings"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
)

type appleRequest struct {
	IDToken string `json:"id_token"`
	Nonce   string `json:"nonce"`
}

type googleRequest struct {
	IDToken string `json:"id_token"`
}

// SignInApple handles POST /v1/auth/apple.
func (h *Handler) SignInApple(w http.ResponseWriter, r *http.Request) {
	var req appleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	if strings.TrimSpace(req.IDToken) == "" || strings.TrimSpace(req.Nonce) == "" {
		httperr.Write(w, r, httperr.ValidationFailed("id_token and nonce are required"))
		return
	}
	res, err := h.svc.SignInApple(r.Context(), req.IDToken, req.Nonce, deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, toSignInResponse(res))
}

// SignInGoogle handles POST /v1/auth/google.
func (h *Handler) SignInGoogle(w http.ResponseWriter, r *http.Request) {
	var req googleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	if strings.TrimSpace(req.IDToken) == "" {
		httperr.Write(w, r, httperr.ValidationFailed("id_token is required"))
		return
	}
	res, err := h.svc.SignInGoogle(r.Context(), req.IDToken, deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, toSignInResponse(res))
}
