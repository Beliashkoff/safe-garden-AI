package handler

import (
	"net/http"
	"strings"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/httperr"
)

// maxOAuthParamLen bounds the code/state/device_id fields well above any real
// provider value while keeping hostile payloads out of the exchange requests.
const maxOAuthParamLen = 1024

type oauthStartResponse struct {
	State         string `json:"state"`
	CodeChallenge string `json:"code_challenge,omitempty"`
	AuthURL       string `json:"auth_url,omitempty"`
}

type yandexCompleteRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type vkCompleteRequest struct {
	Code     string `json:"code"`
	State    string `json:"state"`
	DeviceID string `json:"device_id"`
}

// StartYandex handles POST /v1/auth/yandex/start: returns the authorize URL
// for the system browser plus the attempt state.
func (h *Handler) StartYandex(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.StartYandex(r.Context(), deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, oauthStartResponse{State: res.State, AuthURL: res.AuthURL})
}

// CompleteYandex handles POST /v1/auth/yandex/complete.
func (h *Handler) CompleteYandex(w http.ResponseWriter, r *http.Request) {
	var req yandexCompleteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	if !validOAuthParams(req.Code, req.State) {
		httperr.Write(w, r, httperr.ValidationFailed("code and state are required"))
		return
	}
	res, err := h.svc.CompleteYandex(r.Context(), req.Code, req.State, deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, toSignInResponse(res))
}

// StartVK handles POST /v1/auth/vk/start: returns the state and the PKCE
// code_challenge the VK ID SDK needs for its confidential flow.
func (h *Handler) StartVK(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.StartVK(r.Context(), deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, oauthStartResponse{State: res.State, CodeChallenge: res.CodeChallenge})
}

// CompleteVK handles POST /v1/auth/vk/complete.
func (h *Handler) CompleteVK(w http.ResponseWriter, r *http.Request) {
	var req vkCompleteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		httperr.Write(w, r, err)
		return
	}
	if !validOAuthParams(req.Code, req.State, req.DeviceID) {
		httperr.Write(w, r, httperr.ValidationFailed("code, state and device_id are required"))
		return
	}
	res, err := h.svc.CompleteVK(r.Context(), req.Code, req.State, req.DeviceID, deviceMetaFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, toSignInResponse(res))
}

func validOAuthParams(values ...string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == "" || len(v) > maxOAuthParamLen {
			return false
		}
	}
	return true
}
