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

-- name: ActiveUserWindows :one
-- DAU/WAU/MAU: distinct users who sent a message inside each trailing window.
-- All three read the same 30-day scan; the narrower windows are FILTERed.
SELECT
    COUNT(DISTINCT user_id) FILTER (WHERE created_at >= @day_since)::bigint  AS dau,
    COUNT(DISTINCT user_id) FILTER (WHERE created_at >= @week_since)::bigint AS wau,
    COUNT(DISTINCT user_id)::bigint                                          AS mau
FROM messages
WHERE role = 'user' AND created_at >= @month_since;

-- name: MessageStatusCountsSince :one
-- Terminal-status breakdown of assistant turns; success_rate = complete / (complete+failed+cancelled).
SELECT
    COUNT(*) FILTER (WHERE status = 'complete')::bigint  AS complete,
    COUNT(*) FILTER (WHERE status = 'failed')::bigint    AS failed,
    COUNT(*) FILTER (WHERE status = 'cancelled')::bigint AS cancelled,
    COUNT(*) FILTER (WHERE status = 'pending')::bigint   AS pending
FROM messages
WHERE role = 'assistant' AND created_at >= $1;

-- name: MessageStatusByDay :many
SELECT
    date_trunc('day', created_at, 'UTC')::timestamptz    AS day,
    COUNT(*) FILTER (WHERE status = 'complete')::bigint  AS complete,
    COUNT(*) FILTER (WHERE status = 'failed')::bigint    AS failed,
    COUNT(*) FILTER (WHERE status = 'cancelled')::bigint AS cancelled
FROM messages
WHERE role = 'assistant' AND created_at >= $1
GROUP BY 1
ORDER BY 1;

-- name: FeedbackTotalsSince :one
-- One verdict per (message, user), so up+down equals the number of rated answers.
SELECT
    COUNT(*) FILTER (WHERE value = 'up')::bigint   AS up,
    COUNT(*) FILTER (WHERE value = 'down')::bigint AS down
FROM message_feedback
WHERE created_at >= $1;

-- name: FeedbackByDay :many
SELECT
    date_trunc('day', created_at, 'UTC')::timestamptz AS day,
    COUNT(*) FILTER (WHERE value = 'up')::bigint      AS up,
    COUNT(*) FILTER (WHERE value = 'down')::bigint    AS down
FROM message_feedback
WHERE created_at >= $1
GROUP BY 1
ORDER BY 1;

-- name: LoginsByProviderSince :many
-- Sign-in events by RU provider (406-FZ: yandex / vk / email-OTP).
SELECT action, COUNT(*)::bigint AS logins, COUNT(DISTINCT user_id)::bigint AS users
FROM audit_log
WHERE action IN ('sign_in_yandex', 'sign_in_vk', 'sign_in_email') AND created_at >= $1
GROUP BY action
ORDER BY logins DESC;

-- name: GetDeletionPipeline :one
-- Account-deletion -> media-purge pipeline health. oldest_pending_hours surfaces a
-- stalled cleanup cron (deleted but Object Storage prefix not yet wiped).
SELECT
    COUNT(*) FILTER (WHERE deleted_at IS NOT NULL)::bigint                             AS deleted_total,
    COUNT(*) FILTER (WHERE deleted_at IS NOT NULL AND media_purged_at IS NULL)::bigint AS purge_pending,
    COALESCE(
        EXTRACT(EPOCH FROM (NOW() - MIN(deleted_at) FILTER (WHERE media_purged_at IS NULL))) / 3600,
        0
    )::float8 AS oldest_pending_hours
FROM users;

-- name: TopCostUsersSince :many
-- Most expensive users by Claude spend (taps excluded). user_id is masked to a hex
-- prefix in the usecase before it leaves the backend (CLAUDE.md invariant #3/#10).
SELECT
    user_id,
    COUNT(*)::bigint                       AS requests,
    COALESCE(SUM(tokens_in), 0)::bigint    AS tokens_in,
    COALESCE(SUM(tokens_out), 0)::bigint   AS tokens_out,
    COALESCE(SUM(cost_usd), 0)::numeric    AS cost_usd
FROM usage_log
WHERE created_at >= $1 AND endpoint NOT LIKE 'fertilizer_tap:%'
GROUP BY user_id
ORDER BY cost_usd DESC
LIMIT 20;

-- name: ErrorsByRouteSince :many
SELECT route, status, COUNT(*)::bigint AS count
FROM admin_error_events
WHERE created_at >= $1
GROUP BY route, status
ORDER BY count DESC
LIMIT 20;

-- name: ErrorsByDaySince :many
SELECT date_trunc('day', created_at, 'UTC')::timestamptz AS day, COUNT(*)::bigint AS count
FROM admin_error_events
WHERE created_at >= $1
GROUP BY 1
ORDER BY 1;

-- name: SumCostBetween :one
-- Bounded-window Claude spend for month-to-date vs previous-month comparison.
SELECT COALESCE(SUM(cost_usd), 0)::numeric AS cost_usd
FROM usage_log
WHERE created_at >= @from_ts AND created_at < @to_ts AND endpoint NOT LIKE 'fertilizer_tap:%';
