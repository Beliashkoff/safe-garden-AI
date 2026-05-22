-- +goose Up
CREATE TABLE refresh_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   BYTEA NOT NULL,
    device_id    TEXT,
    user_agent   TEXT,
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- O(log n) lookup by sha256 hash; UNIQUE also prevents accidental hash reuse.
CREATE UNIQUE INDEX refresh_tokens_token_hash_idx ON refresh_tokens (token_hash);

-- Partial index for fast "list active sessions for user X" — the only common
-- read pattern besides the hash lookup.
CREATE INDEX refresh_tokens_user_active_idx
    ON refresh_tokens (user_id)
    WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
