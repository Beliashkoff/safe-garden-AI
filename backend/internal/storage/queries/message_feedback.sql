-- name: UpsertMessageFeedback :exec
-- PUT /v1/messages/:id/feedback — owner-scoped (user_id in the key). Idempotent:
-- re-voting the same value just refreshes updated_at.
INSERT INTO message_feedback (message_id, user_id, value)
VALUES ($1, $2, $3)
ON CONFLICT (message_id, user_id) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();

-- name: DeleteMessageFeedback :exec
-- Clears the caller's verdict on a message. Idempotent (no-op if absent).
DELETE FROM message_feedback WHERE message_id = $1 AND user_id = $2;
