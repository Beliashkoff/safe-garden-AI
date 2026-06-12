-- +goose Up
-- 406-FZ compliance: foreign OAuth (Apple/Google) is removed; sign-in is
-- Yandex ID, VK ID or email OTP only. Existing apple_sub/google_sub identities
-- are dropped — affected accounts remain reachable via email OTP (verified
-- emails were stored at link time).
ALTER TABLE users ADD COLUMN yandex_sub TEXT UNIQUE;
ALTER TABLE users ADD COLUMN vk_sub TEXT UNIQUE;
ALTER TABLE users DROP COLUMN apple_sub;
ALTER TABLE users DROP COLUMN google_sub;

-- Server-side OAuth attempt state: the CSRF state (stored hashed, like
-- refresh tokens) and the PKCE code_verifier, which never leaves the backend.
-- Rows are single-use (consumed via DELETE ... RETURNING) and short-lived;
-- expired rows are swept opportunistically on each new attempt.
CREATE TABLE oauth_states (
    state_hash    BYTEA PRIMARY KEY,
    provider      TEXT NOT NULL,
    code_verifier TEXT NOT NULL,
    ip            TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ NOT NULL
);

-- Supports the per-IP active-attempt cap (anti-flood on /start).
CREATE INDEX oauth_states_ip_idx ON oauth_states (ip) WHERE ip IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS oauth_states;
ALTER TABLE users ADD COLUMN apple_sub TEXT UNIQUE;
ALTER TABLE users ADD COLUMN google_sub TEXT UNIQUE;
ALTER TABLE users DROP COLUMN yandex_sub;
ALTER TABLE users DROP COLUMN vk_sub;
