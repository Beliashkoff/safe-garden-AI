-- name: CreateMessage :one
INSERT INTO messages (conversation_id, user_id, role, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetMessageByID :one
SELECT * FROM messages WHERE id = $1;

-- name: ListRecentMessages :many
-- First page of history (latest-first). The next cursor is the (created_at, id)
-- of the last returned row.
SELECT * FROM messages
WHERE conversation_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2;

-- name: ListMessagesBefore :many
-- Keyset page strictly older than the cursor. The row-value comparison rides
-- the (conversation_id, created_at, id) index and is stable across created_at
-- ties. Explicit casts so sqlc types the id cursor as uuid, not timestamptz.
SELECT * FROM messages
WHERE conversation_id = sqlc.arg('conversation_id')
  AND (created_at, id) < (sqlc.arg('before_created_at')::timestamptz, sqlc.arg('before_id')::uuid)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: UpdateMessageStatus :exec
UPDATE messages SET status = $2 WHERE id = $1;

-- name: CompleteMessage :exec
-- Finalises an assistant message: status + token counts (from the worker's SSE
-- `usage` event, ARCH §11.3).
UPDATE messages SET status = 'complete', tokens_in = $2, tokens_out = $3 WHERE id = $1;

-- name: DeleteMessage :execrows
-- DELETE /v1/messages/:id — owner-scoped (user_id in WHERE). execrows lets the
-- handler distinguish 404 (0 rows) from 204 (1 row). Blocks cascade.
DELETE FROM messages WHERE id = $1 AND user_id = $2;
