-- +goose Up
-- Records why an assistant turn ended in status='failed' (upstream_error,
-- tool_loop_exhausted, ...). Until now the failure code lived only in the
-- Prometheus claude_request_errors_total{code} counter, invisible to the
-- operator; messages.status='failed' kept the fact but not the cause. NULL for
-- non-failed messages and for rows written before this migration.
ALTER TABLE messages ADD COLUMN fail_code TEXT;

-- +goose Down
ALTER TABLE messages DROP COLUMN IF EXISTS fail_code;
