package handler

import (
	"net/http"
	"time"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/ctxkey"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
	uploaduc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/upload"
)

type presignRequest struct {
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type presignResponse struct {
	URL       string            `json:"url"`
	Key       string            `json:"key"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type presignViewRequest struct {
	StorageKey string `json:"storage_key"`
}

type presignViewResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PostPresign handles POST /v1/uploads/presign — issues a presigned PUT URL for
// a photo upload (ARCH §4.3). The client PUTs the file directly to object
// storage, then references the returned key in POST /v1/messages.
func (h *Handler) PostPresign(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxkey.UserID(r.Context())
	if !ok {
		httperr.Write(w, r, httperr.Unauthorized("authentication required"))
		return
	}
	var req presignRequest
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	out, err := h.upload.Presign(r.Context(), userID, uploaduc.PresignInput{
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, presignResponse{
		URL:       out.URL,
		Key:       out.Key,
		Headers:   out.Headers,
		ExpiresAt: out.ExpiresAt,
	})
}

// PostPresignView handles POST /v1/uploads/view — issues a short-lived presigned
// GET URL for an owned photo so the client can display it (e.g. history after a
// reinstall). Ownership is enforced in the usecase by the owner-scoped key
// prefix. The storage key is passed in the body (it contains slashes).
func (h *Handler) PostPresignView(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxkey.UserID(r.Context())
	if !ok {
		httperr.Write(w, r, httperr.Unauthorized("authentication required"))
		return
	}
	var req presignViewRequest
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	out, err := h.upload.PresignView(r.Context(), userID, req.StorageKey)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, presignViewResponse{
		URL:       out.URL,
		ExpiresAt: out.ExpiresAt,
	})
}
