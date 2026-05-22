-- name: CreateEmailCode :one
INSERT INTO email_codes (email, code_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetActiveEmailCode :one
-- Returns the most recent unused, unexpired code for the email. Older codes
-- become irrelevant the moment a newer code is issued.
SELECT * FROM email_codes
WHERE email = $1
  AND used_at IS NULL
  AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: IncrementEmailCodeAttempts :one
-- RETURNING attempts lets the caller atomically enforce the ≤5 attempts cap.
UPDATE email_codes
SET attempts = attempts + 1
WHERE id = $1
RETURNING attempts;

-- name: MarkEmailCodeUsed :exec
UPDATE email_codes SET used_at = NOW() WHERE id = $1;

-- name: CountRecentEmailCodes :one
-- DB-level baseline for the ≤3-requests/hour/email rate limit. Redis adds a
-- faster check in front of this when it lands (stage 2.3).
SELECT COUNT(*) FROM email_codes
WHERE email = $1 AND created_at > NOW() - INTERVAL '1 hour';
