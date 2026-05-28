-- name: CreateMessageBlock :one
INSERT INTO message_blocks (message_id, order_index, type, content_text, storage_key, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListBlocksByMessageIDs :many
-- Batch-loads blocks for a page of messages (avoids N+1). Caller groups by
-- message_id; rows arrive ordered within each message by order_index.
SELECT * FROM message_blocks
WHERE message_id = ANY($1::uuid[])
ORDER BY message_id, order_index;
