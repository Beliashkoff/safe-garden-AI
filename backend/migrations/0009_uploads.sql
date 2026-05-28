-- +goose Up
CREATE TABLE uploads (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    storage_key  TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL,
    size_bytes   BIGINT NOT NULL,
    used         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX uploads_user_created_idx ON uploads (user_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS uploads;
