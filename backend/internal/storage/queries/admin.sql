-- Admin panel queries: operator account, sessions, one-time codes, audit log
-- and the server error feed.

-- name: CountAdmins :one
SELECT COUNT(*) FROM admin_users;

-- name: GetAdminByEmail :one
SELECT * FROM admin_users WHERE email = $1;

-- name: GetAdminByID :one
SELECT * FROM admin_users WHERE id = $1;

-- name: CreateAdmin :one
INSERT INTO admin_users (email, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateAdminPassword :exec
UPDATE admin_users
SET password_hash = $2, updated_at = NOW()
WHERE id = $1;

-- name: TouchAdminLogin :exec
UPDATE admin_users SET last_login_at = NOW() WHERE id = $1;

-- name: CreateAdminSession :one
INSERT INTO admin_sessions (admin_id, token_hash, ip, user_agent, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAdminSessionByHash :one
SELECT * FROM admin_sessions
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW();

-- name: TouchAdminSession :exec
-- Sliding TTL: bump both the activity stamp and the expiry on use.
UPDATE admin_sessions
SET last_seen_at = NOW(), expires_at = $2
WHERE id = $1;

-- name: RevokeAdminSession :exec
UPDATE admin_sessions SET revoked_at = NOW() WHERE id = $1;

-- name: RevokeAllAdminSessions :exec
UPDATE admin_sessions
SET revoked_at = NOW()
WHERE admin_id = $1 AND revoked_at IS NULL;

-- name: RevokeOtherAdminSessions :exec
UPDATE admin_sessions
SET revoked_at = NOW()
WHERE admin_id = $1 AND id <> $2 AND revoked_at IS NULL;

-- name: ListActiveAdminSessions :many
SELECT * FROM admin_sessions
WHERE admin_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
ORDER BY last_seen_at DESC;

-- name: DeleteExpiredAdminSessions :execrows
DELETE FROM admin_sessions WHERE expires_at < NOW() - INTERVAL '7 days';

-- name: CreateAdminCode :one
INSERT INTO admin_codes (email, purpose, code_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetActiveAdminCode :one
SELECT * FROM admin_codes
WHERE email = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: IncrementAdminCodeAttempts :one
UPDATE admin_codes SET attempts = attempts + 1 WHERE id = $1 RETURNING attempts;

-- name: MarkAdminCodeUsed :exec
UPDATE admin_codes SET used_at = NOW() WHERE id = $1;

-- name: CountRecentAdminCodes :one
SELECT COUNT(*) FROM admin_codes
WHERE email = $1 AND purpose = $2 AND created_at >= $3;

-- name: DeleteExpiredAdminCodes :execrows
DELETE FROM admin_codes WHERE expires_at < NOW() - INTERVAL '1 day';

-- name: InsertAdminAudit :exec
INSERT INTO admin_audit_log (admin_id, action, entity, entity_id, details, ip)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListAdminAudit :many
SELECT * FROM admin_audit_log
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountRecentFailedAdminLogins :one
-- DB-baseline brute-force guard: failed logins from one IP inside the window.
SELECT COUNT(*) FROM admin_audit_log
WHERE action = 'admin_login_failed' AND ip = $1 AND created_at >= $2;

-- name: CountRecentFailedAdminLoginsGlobal :one
-- Account-global guard (IP-independent): closes the per-IP bypass when an
-- attacker rotates spoofed X-Forwarded-For values. The single-operator panel
-- has one account, so a global cap is acceptable; it self-heals after the window.
SELECT COUNT(*) FROM admin_audit_log
WHERE action = 'admin_login_failed' AND created_at >= $1;

-- name: InsertErrorEvent :exec
INSERT INTO admin_error_events (source, route, method, status, request_id, message)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListErrorEvents :many
SELECT * FROM admin_error_events
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountErrorEventsSince :one
SELECT COUNT(*) FROM admin_error_events WHERE created_at >= $1;

-- name: DeleteOldErrorEvents :execrows
DELETE FROM admin_error_events WHERE created_at < $1;
