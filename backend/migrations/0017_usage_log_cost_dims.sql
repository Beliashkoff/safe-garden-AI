-- +goose Up
-- Cost/FinOps instrumentation for usage_log. All columns nullable: NULL on rows
-- written before this migration and on rows where the dimension does not apply.
--   * input_uncached_tokens / cache_write_tokens / cache_read_tokens — splits the
--     input token count by cache tier so prompt-cache hit-rate and savings are
--     measurable (confirms the caching fix fb01679 actually saves money). Until
--     now only the collapsed total (tokens_in) was stored.
--   * model — the model id requested for the turn, to compare cost across Opus
--     versions objectively instead of by git history.
--   * duration_ms — voice transcription length for SpeechKit (ruble) volume; set
--     only on synthetic endpoint='transcribe' rows.
ALTER TABLE usage_log
    ADD COLUMN input_uncached_tokens INT,
    ADD COLUMN cache_write_tokens    INT,
    ADD COLUMN cache_read_tokens     INT,
    ADD COLUMN model                 TEXT,
    ADD COLUMN duration_ms           INT;

-- +goose Down
ALTER TABLE usage_log
    DROP COLUMN IF EXISTS duration_ms,
    DROP COLUMN IF EXISTS model,
    DROP COLUMN IF EXISTS cache_read_tokens,
    DROP COLUMN IF EXISTS cache_write_tokens,
    DROP COLUMN IF EXISTS input_uncached_tokens;
