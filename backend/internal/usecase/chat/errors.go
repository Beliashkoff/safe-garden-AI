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
	// ErrUploadNotFound — referenced image_ref/audio_ref upload is missing or not
	// owned by the caller (or its key is malformed).
	ErrUploadNotFound = errors.New("chat: upload not found")
	// ErrAudioTooLong — voice recording exceeds the duration cap (60s).
	ErrAudioTooLong = errors.New("chat: audio too long")
	// ErrTranscriptionEmpty — transcription produced no text (no speech detected).
	ErrTranscriptionEmpty = errors.New("chat: transcription empty")
	// ErrTranscriptionFailed — the transcription pipeline failed (download /
	// convert / recognize). Treated as a transient/upstream failure.
	ErrTranscriptionFailed = errors.New("chat: transcription failed")
)
