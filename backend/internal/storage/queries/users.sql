-- name: CreateUser :one
INSERT INTO users (email, email_verified, apple_sub, google_sub, display_name, locale)
VALUES ($1, $2, $3, $4, $5, COALESCE(NULLIF($6, ''), 'ru'))
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByAppleSub :one
SELECT * FROM users WHERE apple_sub = $1 AND deleted_at IS NULL;

-- name: GetUserByGoogleSub :one
SELECT * FROM users WHERE google_sub = $1 AND deleted_at IS NULL;

-- name: LinkAppleSub :one
UPDATE users SET apple_sub = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: LinkGoogleSub :one
UPDATE users SET google_sub = $2, updated_at = NOW()
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
-- email or OAuth subject later. Apple/Google review explicitly requires
-- account deletion to free up identifiers.
UPDATE users
SET deleted_at = NOW(),
    email = NULL,
    apple_sub = NULL,
    google_sub = NULL,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
