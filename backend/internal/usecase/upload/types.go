// Package upload is the uploads usecase: it validates an upload request, issues
// a presigned PUT URL to object storage, and records the pending upload so the
// chat usecase can later verify ownership (ARCH §4.3, §5).
package upload

import "time"

// PresignInput is the decoded POST /v1/uploads/presign body.
type PresignInput struct {
	ContentType string
	SizeBytes   int64
}

// PresignOutput is returned to the client; it PUTs the file to URL (with the
// given Headers) before Key's expiry.
type PresignOutput struct {
	URL       string
	Key       string
	Headers   map[string]string
	ExpiresAt time.Time
}

// ViewOutput is returned to the client to display a stored photo: a presigned
// GET URL valid until ExpiresAt.
type ViewOutput struct {
	URL       string
	ExpiresAt time.Time
}
