package handler

import (
	"net/http"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
)

type emailRequestBody struct {
	Email string `json:"email"`
}

type emailVerifyBody struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// RequestOTP handles POST /v1/auth/email/request. Always returns 204 on success
// regardless of whether the address is registered.
func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req emailRequestBody
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	if err := h.svc.RequestOTP(r.Context(), req.Email, localeFrom(r)); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// VerifyOTP handles POST /v1/auth/email/verify.
func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req emailVerifyBody
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	res, err := h.svc.VerifyOTP(r.Context(), req.Email, req.Code, deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, toSignInResponse(res))
}
