-- name: InsertAuditLog :exec
-- user_id is nullable for system-level events (e.g. failed verification of an
-- id_token before any user could be resolved).
INSERT INTO audit_log (user_id, action, details, ip)
VALUES ($1, $2, $3, $4);

-- name: ListAuditByUser :many
SELECT * FROM audit_log
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;
