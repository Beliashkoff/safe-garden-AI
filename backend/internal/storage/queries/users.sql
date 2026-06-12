-- name: CreateUser :one
INSERT INTO users (email, email_verified, yandex_sub, vk_sub, display_name, locale)
VALUES ($1, $2, $3, $4, $5, COALESCE(NULLIF($6, ''), 'ru'))
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByYandexSub :one
SELECT * FROM users WHERE yandex_sub = $1 AND deleted_at IS NULL;

-- name: GetUserByVKSub :one
SELECT * FROM users WHERE vk_sub = $1 AND deleted_at IS NULL;

-- name: LinkYandexSub :one
UPDATE users SET yandex_sub = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: LinkVKSub :one
UPDATE users SET vk_sub = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SetUserEmail :one
UPDATE users SET email = $2, email_verified = $3, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: MarkEmailVerified :exec
UPDATE users SET email_verified = TRUE, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteUser :exec
-- Null out unique identifiers so the user can re-register with the same
-- email or OAuth subject later. Store review (and 152-FZ erasure requests)
-- require account deletion to free up identifiers.
UPDATE users
SET deleted_at = NOW(),
    email = NULL,
    yandex_sub = NULL,
    vk_sub = NULL,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteUserConversations :exec
-- Hard-deletes the user's conversation; cascades to messages and message_blocks
-- (ARCH §6.3 / SPEC F9). Called inside DeleteAccount's transaction.
DELETE FROM conversations WHERE user_id = $1;

-- name: DeleteUserUploads :exec
-- Hard-deletes the user's upload rows. The objects in Object Storage are removed
-- asynchronously by the cleanup job (prefix u/{user_id}/).
DELETE FROM uploads WHERE user_id = $1;

-- name: ListUsersPendingMediaPurge :many
-- Cleanup work-list: deleted accounts whose Object Storage media has not been
-- purged yet. Oldest first; bounded by $1.
SELECT id FROM users
WHERE deleted_at IS NOT NULL AND media_purged_at IS NULL
ORDER BY deleted_at
LIMIT $1;

-- name: MarkUserMediaPurged :exec
-- Marks the user's media prefix as purged so the cleanup job skips it next run.
UPDATE users SET media_purged_at = NOW() WHERE id = $1;
