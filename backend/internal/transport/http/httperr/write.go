package httperr

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// envelope is the §4.7 wire shape: { error: {...}, request_id: "..." }.
type envelope struct {
	Error     *Error `json:"error"`
	RequestID string `json:"request_id"`
}

// Write serializes err as the §4.7 envelope and writes it with err.HTTPStatus.
// If err is not an *Error it is treated as an internal error: the original is
// logged (with request_id) and a generic 500 is returned so internal details
// never leak to clients. Server-side (5xx) errors are always logged.
func Write(w http.ResponseWriter, r *http.Request, err error) {
	reqID := middleware.GetReqID(r.Context())

	var he *Error
	if !errors.As(err, &he) {
		he = Internal("internal error")
	}

	if he.HTTPStatus >= 500 {
		slog.ErrorContext(r.Context(), "request failed",
			"request_id", reqID,
			"status", he.HTTPStatus,
			"code", string(he.Code),
			"err", err.Error(),
		)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(he.HTTPStatus)
	if encErr := json.NewEncoder(w).Encode(envelope{Error: he, RequestID: reqID}); encErr != nil {
		slog.ErrorContext(r.Context(), "failed to encode error response",
			"request_id", reqID, "err", encErr.Error())
	}
}
