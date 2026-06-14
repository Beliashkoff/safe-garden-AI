-- Aggregates for the admin panel dashboard. All read-only. Day bucketing is
-- pinned to UTC via date_trunc(field, ts, 'UTC') (PG14+) so it does not depend
-- on the server session TimeZone and agrees with the Go-side UTC cutoffs.

-- name: CountActiveUsers :one
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;

-- name: CountUsersCreatedSince :one
SELECT COUNT(*) FROM users WHERE created_at >= $1;

-- name: CountUserMessages :one
SELECT COUNT(*) FROM messages WHERE role = 'user';

-- name: CountUserMessagesSince :one
SELECT COUNT(*) FROM messages WHERE role = 'user' AND created_at >= $1;

-- name: SumUsageSince :one
SELECT
    COALESCE(SUM(tokens_in), 0)::bigint    AS tokens_in,
    COALESCE(SUM(tokens_out), 0)::bigint   AS tokens_out,
    COALESCE(SUM(cost_usd), 0)::numeric    AS cost_usd
FROM usage_log
WHERE created_at >= $1 AND endpoint NOT LIKE 'fertilizer_tap:%';

-- name: CountFertilizerTapsSince :one
SELECT COUNT(*) FROM usage_log
WHERE endpoint LIKE 'fertilizer_tap:%' AND created_at >= $1;

-- name: UsersByDay :many
SELECT date_trunc('day', created_at, 'UTC')::timestamptz AS day, COUNT(*) AS count
FROM users
WHERE created_at >= $1
GROUP BY 1
ORDER BY 1;

-- name: MessagesByDay :many
SELECT date_trunc('day', created_at, 'UTC')::timestamptz AS day, COUNT(*) AS count
FROM messages
WHERE role = 'user' AND created_at >= $1
GROUP BY 1
ORDER BY 1;

-- name: UsageByDay :many
SELECT
    date_trunc('day', created_at, 'UTC')::timestamptz AS day,
    COALESCE(SUM(tokens_in), 0)::bigint             AS tokens_in,
    COALESCE(SUM(tokens_out), 0)::bigint            AS tokens_out,
    COALESCE(SUM(cost_usd), 0)::numeric             AS cost_usd
FROM usage_log
WHERE created_at >= $1 AND endpoint NOT LIKE 'fertilizer_tap:%'
GROUP BY 1
ORDER BY 1;

-- name: TopFertilizerTaps :many
SELECT split_part(endpoint, ':', 2) AS slug, COUNT(*) AS taps
FROM usage_log
WHERE endpoint LIKE 'fertilizer_tap:%' AND created_at >= $1
GROUP BY 1
ORDER BY 2 DESC
LIMIT 10;
