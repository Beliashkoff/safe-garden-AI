-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, device_id, user_agent, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetRefreshTokenByHash :one
-- Intentionally returns the row even when revoked or expired so the caller
-- can implement reuse-detection (presenting a revoked token revokes the
-- whole family).
SELECT * FROM refresh_tokens WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE id = $1 AND revoked_at IS NULL;

-- name: TouchRefreshToken :exec
UPDATE refresh_tokens SET last_used_at = NOW() WHERE id = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: DeleteExpiredRefreshTokens :exec
-- Periodic cleanup job (wired in a later stage). Keeps recently expired rows
-- for a week to aid debugging reuse incidents.
DELETE FROM refresh_tokens WHERE expires_at < NOW() - INTERVAL '7 days';
