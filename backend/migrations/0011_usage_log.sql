-- +goose Up
CREATE TABLE usage_log (
    id         BIGSERIAL PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint   TEXT NOT NULL,
    tokens_in  INT,
    tokens_out INT,
    cost_usd   NUMERIC(10,6),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX usage_log_user_created_idx ON usage_log (user_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS usage_log;
