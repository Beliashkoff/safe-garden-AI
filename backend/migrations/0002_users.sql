-- +goose Up
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email          CITEXT UNIQUE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    apple_sub      TEXT UNIQUE,
    google_sub     TEXT UNIQUE,
    display_name   TEXT,
    locale         TEXT NOT NULL DEFAULT 'ru',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

-- +goose Down
DROP TABLE IF EXISTS users;
