-- name: CreateOAuthState :exec
INSERT INTO oauth_states (state_hash, provider, code_verifier, ip, expires_at)
VALUES ($1, $2, $3, $4, $5);

-- name: ConsumeOAuthState :one
-- Single-use: the row is removed atomically on first presentation, so a
-- replayed state (or one stolen from the redirect) fails the second time.
DELETE FROM oauth_states
WHERE state_hash = $1 AND provider = $2 AND expires_at > NOW()
RETURNING *;

-- name: CountActiveOAuthStatesByIP :one
-- Anti-flood cap for POST /auth/{provider}/start.
SELECT COUNT(*) FROM oauth_states WHERE ip = $1 AND expires_at > NOW();

-- name: DeleteExpiredOAuthStates :exec
-- Opportunistic sweep, called best-effort on each new attempt.
DELETE FROM oauth_states WHERE expires_at <= NOW();
