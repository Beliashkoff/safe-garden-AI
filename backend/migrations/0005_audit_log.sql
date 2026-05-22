-- +goose Up
CREATE TABLE audit_log (
    id         BIGSERIAL PRIMARY KEY,
    user_id    UUID,
    action     TEXT NOT NULL,
    details    JSONB,
    ip         INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Supports ListAuditByUser (recent events for a user, descending by time).
CREATE INDEX audit_log_user_created_idx
    ON audit_log (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_log;
