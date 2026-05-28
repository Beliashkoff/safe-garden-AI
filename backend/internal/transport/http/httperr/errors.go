// Package httperr defines the single error envelope used by the HTTP transport
// (ARCHITECTURE.md §4.7) and helpers to write it. Keeping construction and
// serialization here means every endpoint emits an identical shape, and new
// endpoints in later stages cannot drift the contract.
package httperr

import "fmt"

// Code is the machine-readable error code returned to clients. The set is
// closed and mirrors ARCHITECTURE.md §4.7 — the mobile app switches on these.
type Code string

const (
	CodeUnauthorized     Code = "unauthorized"
	CodeForbidden        Code = "forbidden"
	CodeValidationFailed Code = "validation_failed"
	CodeNotFound         Code = "not_found"
	CodeRateLimited      Code = "rate_limited"
	CodePayloadTooLarge  Code = "payload_too_large"
	CodeUnsupportedMedia Code = "unsupported_media_type"
	CodeInternalError    Code = "internal_error"
)

// Error is a transport-level error carrying the HTTP status plus the §4.7 body
// fields. It implements error so it can flow through normal return paths.
type Error struct {
	HTTPStatus int            `json:"-"`
	Code       Code           `json:"code"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("httperr: %d %s: %s", e.HTTPStatus, e.Code, e.Message)
}

// WithDetail returns a copy of e with one detail field set. Useful for adding
// {"field": "email"} to a validation error without mutating a shared value.
func (e *Error) WithDetail(key string, val any) *Error {
	cp := *e
	cp.Details = map[string]any{}
	for k, v := range e.Details {
		cp.Details[k] = v
	}
	cp.Details[key] = val
	return &cp
}

func Unauthorized(msg string) *Error {
	return &Error{HTTPStatus: 401, Code: CodeUnauthorized, Message: msg}
}

func Forbidden(msg string) *Error {
	return &Error{HTTPStatus: 403, Code: CodeForbidden, Message: msg}
}

func ValidationFailed(msg string) *Error {
	return &Error{HTTPStatus: 400, Code: CodeValidationFailed, Message: msg}
}

func NotFound(msg string) *Error {
	return &Error{HTTPStatus: 404, Code: CodeNotFound, Message: msg}
}

func RateLimited(msg string) *Error {
	return &Error{HTTPStatus: 429, Code: CodeRateLimited, Message: msg}
}

func PayloadTooLarge(msg string) *Error {
	return &Error{HTTPStatus: 413, Code: CodePayloadTooLarge, Message: msg}
}

func UnsupportedMedia(msg string) *Error {
	return &Error{HTTPStatus: 415, Code: CodeUnsupportedMedia, Message: msg}
}

func Internal(msg string) *Error {
	return &Error{HTTPStatus: 500, Code: CodeInternalError, Message: msg}
}
