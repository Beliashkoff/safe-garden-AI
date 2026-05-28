package chat

import "errors"

var (
	// ErrEmptyContent — no usable text in the request.
	ErrEmptyContent = errors.New("chat: empty content")
	// ErrUnsupportedBlock — a non-text block in a stage where only text is allowed.
	ErrUnsupportedBlock = errors.New("chat: unsupported content block")
	// ErrTextTooLarge — combined text exceeds the payload cap.
	ErrTextTooLarge = errors.New("chat: text too large")
	// ErrRateLimited — per-user message rate exceeded.
	ErrRateLimited = errors.New("chat: rate limited")
	// ErrMessageNotFound — message missing or not owned by the caller.
	ErrMessageNotFound = errors.New("chat: message not found")
	// ErrBadCursor — pagination cursor is malformed.
	ErrBadCursor = errors.New("chat: bad cursor")
)
