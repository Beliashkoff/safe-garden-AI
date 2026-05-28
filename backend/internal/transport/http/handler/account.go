package handler

import (
	"net/http"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/ctxkey"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
)

type accountResponse struct {
	User userDTO `json:"user"`
}

// GetAccount handles GET /v1/account. Behind RequireAuth.
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxkey.UserID(r.Context())
	if !ok {
		httperr.Write(w, r, httperr.Unauthorized("authentication required"))
		return
	}
	view, err := h.svc.GetAccount(r.Context(), userID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, accountResponse{User: toUserDTO(view)})
}

// DeleteAccount handles DELETE /v1/account. Behind RequireAuth. The user_id is
// taken from the verified token only — never from the body or path.
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxkey.UserID(r.Context())
	if !ok {
		httperr.Write(w, r, httperr.Unauthorized("authentication required"))
		return
	}
	if err := h.svc.DeleteAccount(r.Context(), userID, deviceMetaFrom(r)); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
