-- +goose Up
CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('user','assistant','system')),
    status          TEXT NOT NULL CHECK (status IN ('pending','complete','cancelled','failed')) DEFAULT 'complete',
    tokens_in       INT,
    tokens_out      INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Latest-first keyset pagination of history (created_at, id) plus ownership scans.
-- user_id is denormalised here so ownership checks need no join (invariant #2).
CREATE INDEX messages_conversation_created_idx ON messages (conversation_id, created_at, id);

-- +goose Down
DROP TABLE IF EXISTS messages;
