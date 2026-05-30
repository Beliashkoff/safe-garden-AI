package upload

import "errors"

var (
	// ErrUnsupportedType — content type not in the image whitelist (ARCH §8.2).
	ErrUnsupportedType = errors.New("upload: unsupported content type")
	// ErrTooLarge — declared size exceeds the per-image cap.
	ErrTooLarge = errors.New("upload: file too large")
	// ErrInvalidSize — non-positive declared size.
	ErrInvalidSize = errors.New("upload: invalid size")
)
