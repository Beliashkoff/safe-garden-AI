-- name: InsertUsage :exec
INSERT INTO usage_log (
    user_id, endpoint, tokens_in, tokens_out, cost_usd,
    input_uncached_tokens, cache_write_tokens, cache_read_tokens, model, duration_ms
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: SumUserTokensSince :one
-- Backs per-user daily token limits and budget alerts (ARCH §13 cost risk; wired
-- in Stage 2.3).
SELECT
    COALESCE(SUM(tokens_in), 0)::bigint AS tokens_in,
    COALESCE(SUM(tokens_out), 0)::bigint AS tokens_out
FROM usage_log
WHERE user_id = $1 AND created_at >= $2;
