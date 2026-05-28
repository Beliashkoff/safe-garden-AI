-- +goose Up
CREATE TABLE message_blocks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id   UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    order_index  INT NOT NULL,
    type         TEXT NOT NULL CHECK (type IN ('text','image','audio','transcription','tool_use','tool_result','fertilizer_card')),
    content_text TEXT,
    storage_key  TEXT,
    metadata     JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX message_blocks_message_order_idx ON message_blocks (message_id, order_index);

-- +goose Down
DROP TABLE IF EXISTS message_blocks;
