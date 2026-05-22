-- +goose Up
CREATE TABLE email_codes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      CITEXT NOT NULL,
    code_hash  BYTEA NOT NULL,
    attempts   INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Lookup index: GetActiveEmailCode filters by email and expires_at.
CREATE INDEX email_codes_email_expires_idx ON email_codes (email, expires_at);

-- +goose Down
DROP TABLE IF EXISTS email_codes;
