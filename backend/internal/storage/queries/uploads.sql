-- name: CreateUpload :one
INSERT INTO uploads (user_id, storage_key, content_type, size_bytes)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUploadByStorageKey :one
-- Used by POST /v1/messages to verify the caller owns the referenced storage_key.
SELECT * FROM uploads WHERE storage_key = $1;

-- name: MarkUploadUsed :exec
UPDATE uploads SET used = TRUE WHERE storage_key = $1;

-- name: ListUnusedUploadsBefore :many
-- GC candidates: presigned-but-never-attached uploads older than the cutoff.
SELECT * FROM uploads WHERE used = FALSE AND created_at < $1;
