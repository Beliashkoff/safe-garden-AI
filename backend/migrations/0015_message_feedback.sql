-- +goose Up

-- Per-user thumbs up/down on an assistant message (ROADMAP §5.x feedback).
-- Owner-scoped via user_id; one verdict per (message, user) so a re-vote is an
-- upsert. value is constrained to the two verdicts; clearing deletes the row.
-- Both FKs cascade so account/message deletion takes feedback with it.
CREATE TABLE message_feedback (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    value      TEXT NOT NULL CHECK (value IN ('up','down')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS message_feedback;
