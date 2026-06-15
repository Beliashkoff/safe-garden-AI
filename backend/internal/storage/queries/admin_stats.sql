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
WHERE created_at >= $1 AND endpoint NOT LIKE 'fertilizer_tap:%' AND endpoint <> 'transcribe'
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

-- ============================================================================
-- Growth analytics (продуктовая аналитика). All read-only over existing tables.
-- ============================================================================

-- name: ActivationSince :one
-- Activation proxy: of users registered in the window, how many asked at least
-- one question, and the median delay from sign-up to that first question. Note:
-- the SPEC §1.4 KPI is "photo in the first session"; input type of the first
-- session is not flagged in the schema, so this approximates with "first question".
WITH firstmsg AS (
    SELECT user_id, MIN(created_at) AS first_at
    FROM messages
    WHERE role = 'user'
    GROUP BY user_id
)
SELECT
    COUNT(*)::bigint        AS signups,
    COUNT(f.user_id)::bigint AS activated,
    COALESCE(EXTRACT(EPOCH FROM percentile_cont(0.5) WITHIN GROUP (
        ORDER BY (f.first_at - u.created_at)
    )), 0)::float8          AS median_seconds
FROM users u
LEFT JOIN firstmsg f ON f.user_id = u.id
WHERE u.created_at >= $1 AND u.deleted_at IS NULL;

-- name: RetentionCohorts :many
-- Weekly sign-up cohorts with D1/D7/D30 return rates. Eligibility gates the
-- denominator so a young cohort that has not yet reached day N is not counted as
-- "churned" (retained / eligible, not retained / size). Computed live; move to a
-- materialized view if message volume grows.
WITH cohort AS (
    SELECT u.id, u.created_at
    FROM users u
    WHERE u.deleted_at IS NULL AND u.created_at >= $1
),
flags AS (
    SELECT
        date_trunc('week', c.created_at, 'UTC') AS week,
        (NOW() >= c.created_at + interval '2 day')  AS elig_d1,
        (NOW() >= c.created_at + interval '8 day')  AS elig_d7,
        (NOW() >= c.created_at + interval '31 day') AS elig_d30,
        EXISTS (SELECT 1 FROM messages m WHERE m.user_id = c.id AND m.role = 'user'
            AND m.created_at >= c.created_at + interval '1 day'
            AND m.created_at <  c.created_at + interval '2 day') AS ret_d1,
        EXISTS (SELECT 1 FROM messages m WHERE m.user_id = c.id AND m.role = 'user'
            AND m.created_at >= c.created_at + interval '7 day'
            AND m.created_at <  c.created_at + interval '8 day') AS ret_d7,
        EXISTS (SELECT 1 FROM messages m WHERE m.user_id = c.id AND m.role = 'user'
            AND m.created_at >= c.created_at + interval '30 day'
            AND m.created_at <  c.created_at + interval '31 day') AS ret_d30
    FROM cohort c
)
SELECT
    week::timestamptz                                         AS week,
    COUNT(*)::bigint                                          AS size,
    COUNT(*) FILTER (WHERE elig_d1)::bigint                   AS d1_eligible,
    COUNT(*) FILTER (WHERE elig_d1 AND ret_d1)::bigint        AS d1_retained,
    COUNT(*) FILTER (WHERE elig_d7)::bigint                   AS d7_eligible,
    COUNT(*) FILTER (WHERE elig_d7 AND ret_d7)::bigint        AS d7_retained,
    COUNT(*) FILTER (WHERE elig_d30)::bigint                  AS d30_eligible,
    COUNT(*) FILTER (WHERE elig_d30 AND ret_d30)::bigint      AS d30_retained
FROM flags
GROUP BY week
ORDER BY week;

-- name: ConversationDepthSince :one
-- Distribution of question count per active user plus avg/median/p90.
WITH per_user AS (
    SELECT user_id, COUNT(*) AS msgs
    FROM messages
    WHERE role = 'user' AND created_at >= $1
    GROUP BY user_id
)
SELECT
    COUNT(*) FILTER (WHERE msgs = 1)::bigint             AS bucket1,
    COUNT(*) FILTER (WHERE msgs BETWEEN 2 AND 4)::bigint AS bucket2_4,
    COUNT(*) FILTER (WHERE msgs BETWEEN 5 AND 9)::bigint AS bucket5_9,
    COUNT(*) FILTER (WHERE msgs >= 10)::bigint           AS bucket10,
    COALESCE(AVG(msgs), 0)::float8                                          AS avg_msgs,
    COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY msgs), 0)::float8  AS median_msgs,
    COALESCE(percentile_cont(0.9) WITHIN GROUP (ORDER BY msgs), 0)::float8  AS p90_msgs
FROM per_user;

-- name: ActivityHeatmapSince :many
-- Question volume by Moscow day-of-week (0=Sun..6=Sat) and hour-of-day.
SELECT
    EXTRACT(DOW  FROM created_at AT TIME ZONE 'Europe/Moscow')::int AS dow,
    EXTRACT(HOUR FROM created_at AT TIME ZONE 'Europe/Moscow')::int AS hour,
    COUNT(*)::bigint                                                AS count
FROM messages
WHERE role = 'user' AND created_at >= $1
GROUP BY 1, 2
ORDER BY 1, 2;

-- name: ActivityByWeekSince :many
-- Weekly seasonal curve over a long horizon: questions and active users.
SELECT
    date_trunc('week', created_at, 'UTC')::timestamptz           AS week,
    COUNT(*) FILTER (WHERE role = 'user')::bigint                AS messages,
    COUNT(DISTINCT user_id) FILTER (WHERE role = 'user')::bigint AS active_users
FROM messages
WHERE created_at >= $1
GROUP BY 1
ORDER BY 1;

-- ============================================================================
-- Answer quality (качество ответов и обратная связь).
-- ============================================================================

-- name: CardImpressionsSince :one
SELECT COUNT(*)::bigint AS impressions
FROM message_blocks
WHERE type = 'fertilizer_card' AND created_at >= $1;

-- name: CardImpressionsBySlugSince :many
-- Per-slug impressions from the {"products":[{"slug":...}]} card metadata.
SELECT (p->>'slug')::text AS slug, COUNT(*)::bigint AS impressions
FROM message_blocks mb,
     LATERAL jsonb_array_elements(mb.metadata->'products') AS p
WHERE mb.type = 'fertilizer_card' AND mb.created_at >= $1
GROUP BY 1;

-- name: TapsBySlugSince :many
-- All slugs (unbounded, unlike TopFertilizerTaps) so CTR can be joined per slug.
SELECT split_part(endpoint, ':', 2) AS slug, COUNT(*)::bigint AS taps
FROM usage_log
WHERE endpoint LIKE 'fertilizer_tap:%' AND created_at >= $1
GROUP BY 1;

-- name: ResponseLengthByVerdictSince :one
-- Average answer length (tokens_out) split by feedback verdict; the question is
-- whether disliked answers are systematically longer or shorter.
SELECT
    COALESCE(AVG(m.tokens_out) FILTER (WHERE f.value = 'up'), 0)::float8   AS avg_up,
    COALESCE(AVG(m.tokens_out) FILTER (WHERE f.value = 'down'), 0)::float8 AS avg_down,
    COALESCE(AVG(m.tokens_out) FILTER (WHERE f.value IS NULL), 0)::float8  AS avg_none,
    COUNT(*) FILTER (WHERE f.value = 'up')::bigint   AS n_up,
    COUNT(*) FILTER (WHERE f.value = 'down')::bigint AS n_down,
    COUNT(*) FILTER (WHERE f.value IS NULL)::bigint  AS n_none
FROM messages m
LEFT JOIN message_feedback f ON f.message_id = m.id
WHERE m.role = 'assistant' AND m.status = 'complete' AND m.created_at >= $1;

-- name: ConversationsWithNegativeFeedbackSince :many
-- Chats accumulating dislikes (1 chat per user in v1, so this flags unhappy users).
SELECT
    m.conversation_id,
    COUNT(*)::bigint               AS downs,
    MAX(f.updated_at)::timestamptz AS last_down
FROM message_feedback f
JOIN messages m ON m.id = f.message_id
WHERE f.value = 'down' AND f.updated_at >= @since
GROUP BY m.conversation_id
HAVING COUNT(*) >= @min_downs
ORDER BY downs DESC, last_down DESC
LIMIT 50;

-- name: FollowupRateSince :one
-- Implicit dissatisfaction proxy: share of assistant answers immediately followed
-- by another user message within 5 minutes (the user had to re-ask).
WITH seq AS (
    SELECT
        m.role,
        m.created_at,
        LEAD(m.role)       OVER w AS next_role,
        LEAD(m.created_at) OVER w AS next_at
    FROM messages m
    WHERE m.created_at >= $1 AND m.role IN ('user', 'assistant')
    WINDOW w AS (PARTITION BY m.conversation_id ORDER BY m.created_at)
)
SELECT
    COUNT(*) FILTER (WHERE role = 'assistant')::bigint AS answers,
    COUNT(*) FILTER (
        WHERE role = 'assistant'
          AND next_role = 'user'
          AND next_at <= created_at + interval '5 minutes'
    )::bigint AS followups
FROM seq;

-- ============================================================================
-- Security & abuse. user_id/email are masked in the usecase before leaving the
-- backend (CLAUDE.md invariant #3); these queries return the raw values only
-- across the storage boundary.
-- ============================================================================

-- name: ListSecurityEvents :many
-- High-signal user security events (session hijack attempts, deletions). Logins
-- are excluded here (aggregated separately on the dashboard) to keep the feed
-- actionable. Allowlist is fixed, not caller-supplied.
SELECT id, user_id, action, ip, created_at
FROM audit_log
WHERE action IN ('refresh_reuse_detected', 'account_deleted', 'account_media_purged')
  AND user_id IS NOT NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: OtpStatsSince :one
-- email-OTP delivery + abuse. delivery_rate = used/issued (silent SMTP failure
-- shows as a drop); exhausted = codes that hit the attempt cap (brute force).
SELECT
    COUNT(*)::bigint                                                       AS issued,
    COUNT(*) FILTER (WHERE used_at IS NOT NULL)::bigint                    AS used,
    COUNT(*) FILTER (WHERE attempts >= 5)::bigint                          AS exhausted,
    COUNT(*) FILTER (WHERE used_at IS NULL AND expires_at < NOW())::bigint AS expired_unused
FROM email_codes
WHERE created_at >= $1;

-- name: TopOtpRequestersSince :many
-- Emails requesting the most codes (mailbox-flood / enumeration). email is
-- masked in the usecase before it leaves the backend.
SELECT email, COUNT(*)::bigint AS codes
FROM email_codes
WHERE created_at >= @since
GROUP BY email
HAVING COUNT(*) >= @min_codes
ORDER BY codes DESC
LIMIT 20;

-- name: SuspiciousIPsSince :many
-- IPs by failed-auth volume: admin-panel brute force + refresh-token reuse. The
-- cutoff is an explicitly-typed CTE so sqlc resolves the param across the UNION.
WITH cutoff AS (SELECT @since::timestamptz AS ts)
SELECT source, ip, COUNT(*)::bigint AS count
FROM (
    SELECT 'admin_login_failed'::text AS source, admin_audit_log.ip
        FROM admin_audit_log, cutoff
        WHERE admin_audit_log.action = 'admin_login_failed'
          AND admin_audit_log.ip IS NOT NULL
          AND admin_audit_log.created_at >= cutoff.ts
    UNION ALL
    SELECT 'refresh_reuse'::text AS source, audit_log.ip
        FROM audit_log, cutoff
        WHERE audit_log.action = 'refresh_reuse_detected'
          AND audit_log.ip IS NOT NULL
          AND audit_log.created_at >= cutoff.ts
) t
GROUP BY source, ip
ORDER BY count DESC
LIMIT 20;

-- ============================================================================
-- Cost / FinOps. Claude rows are endpoint='/v1/messages'; taps and 'transcribe'
-- carry no Claude cost (cost_usd NULL → ignored by the SUMs).
-- ============================================================================

-- name: UnitEconomicsSince :one
SELECT
    COALESCE(SUM(cost_usd), 0)::numeric                                      AS cost_usd,
    COALESCE(SUM(tokens_in), 0)::bigint                                      AS tokens_in,
    COALESCE(SUM(tokens_out), 0)::bigint                                     AS tokens_out,
    COUNT(*) FILTER (WHERE endpoint = '/v1/messages')::bigint                AS messages,
    COUNT(DISTINCT user_id) FILTER (WHERE endpoint = '/v1/messages')::bigint AS users
FROM usage_log
WHERE created_at >= $1 AND endpoint NOT LIKE 'fertilizer_tap:%' AND endpoint <> 'transcribe';

-- name: CacheStatsSince :one
-- Only rows written after migration 0017 carry the cache split; older rows have
-- NULLs and are excluded from the denominator via rows_with_data.
SELECT
    COALESCE(SUM(input_uncached_tokens), 0)::bigint              AS input_uncached,
    COALESCE(SUM(cache_write_tokens), 0)::bigint                 AS cache_write,
    COALESCE(SUM(cache_read_tokens), 0)::bigint                  AS cache_read,
    COUNT(*) FILTER (WHERE cache_read_tokens IS NOT NULL)::bigint AS rows_with_data
FROM usage_log
WHERE created_at >= $1 AND endpoint = '/v1/messages';

-- name: CostByModelSince :many
SELECT
    COALESCE(model, 'до версионирования')::text AS model,
    COALESCE(SUM(cost_usd), 0)::numeric         AS cost_usd,
    COALESCE(SUM(tokens_in), 0)::bigint         AS tokens_in,
    COALESCE(SUM(tokens_out), 0)::bigint        AS tokens_out,
    COUNT(*)::bigint                            AS requests
FROM usage_log
WHERE created_at >= $1 AND endpoint = '/v1/messages'
GROUP BY 1
ORDER BY 2 DESC;

-- name: CostByKindSince :one
SELECT
    COALESCE(SUM(cost_usd) FILTER (WHERE endpoint = '/v1/messages'), 0)::numeric AS claude_cost,
    COUNT(*) FILTER (WHERE endpoint = '/v1/messages')::bigint                    AS claude_calls,
    COUNT(*) FILTER (WHERE endpoint LIKE 'fertilizer_tap:%')::bigint             AS taps,
    COUNT(*) FILTER (WHERE endpoint = 'transcribe')::bigint                      AS transcriptions,
    COALESCE(SUM(duration_ms) FILTER (WHERE endpoint = 'transcribe'), 0)::bigint AS transcribe_ms
FROM usage_log
WHERE created_at >= $1;

-- name: FailCodesSince :many
-- Breakdown of failed assistant turns by reason for the reliability widget.
SELECT COALESCE(fail_code, 'unknown')::text AS fail_code, COUNT(*)::bigint AS count
FROM messages
WHERE role = 'assistant' AND status = 'failed' AND created_at >= $1
GROUP BY 1
ORDER BY 2 DESC;

-- name: ListDownvotedMessages :many
-- Feed of disliked answers for the operator-agronomist to review. Returns message
-- CONTENT (question + answer text): this is a per-request read into the
-- authenticated panel, NOT a log — callers must never write it to slog/Sentry
-- (CLAUDE.md invariant #3).
SELECT
    f.message_id,
    f.updated_at,
    m.conversation_id,
    (SELECT mb.content_text FROM message_blocks mb
        WHERE mb.message_id = m.id AND mb.type = 'text'
        ORDER BY mb.order_index LIMIT 1) AS answer_text,
    (SELECT mb.content_text
        FROM messages um
        JOIN message_blocks mb ON mb.message_id = um.id AND mb.type = 'text'
        WHERE um.conversation_id = m.conversation_id
          AND um.role = 'user' AND um.created_at < m.created_at
        ORDER BY um.created_at DESC, mb.order_index LIMIT 1) AS question_text,
    EXISTS (SELECT 1 FROM message_blocks mb
        WHERE mb.message_id = m.id AND mb.type = 'fertilizer_card') AS had_card
FROM message_feedback f
JOIN messages m ON m.id = f.message_id
WHERE f.value = 'down'
ORDER BY f.updated_at DESC
LIMIT $1 OFFSET $2;
