-- +goose Up
CREATE TABLE conversations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One chat per user in v1 (CLAUDE.md invariant #1). Doubles as the lookup index
-- for GetConversationByUser and the conflict target for get-or-create.
CREATE UNIQUE INDEX conversations_user_id_idx ON conversations (user_id);

-- +goose Down
DROP TABLE IF EXISTS conversations;
