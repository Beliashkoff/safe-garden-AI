-- name: GetConversationByUser :one
SELECT * FROM conversations WHERE user_id = $1;

-- name: GetOrCreateConversation :one
-- Idempotent: one chat per user. The DO UPDATE is a no-op that exists only so
-- RETURNING yields the existing row on conflict — atomic, race-free.
INSERT INTO conversations (user_id) VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET user_id = conversations.user_id
RETURNING *;
