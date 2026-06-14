-- +goose Up

-- Admin panel (ARCH §12 addendum). A single operator account: sign-in is
-- email+password, where the email is pinned to ADMIN_EMAIL from the
-- environment. Setup and password reset are gated by one-time codes delivered
-- over SMTP to that same address only.
CREATE TABLE admin_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         CITEXT UNIQUE NOT NULL,
    password_hash BYTEA NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ
);

-- Opaque session tokens, sha256-hashed like refresh_tokens. Sliding TTL: the
-- middleware extends expires_at on activity.
CREATE TABLE admin_sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id     UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    token_hash   BYTEA UNIQUE NOT NULL,
    ip           INET,
    user_agent   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ
);

CREATE INDEX admin_sessions_admin_idx ON admin_sessions (admin_id);

-- One-time codes for first-time setup and password reset. Mirrors email_codes
-- (bcrypt hash, attempt cap, TTL) with an explicit purpose so a setup code can
-- never be replayed as a reset code.
CREATE TABLE admin_codes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      CITEXT NOT NULL,
    purpose    TEXT NOT NULL CHECK (purpose IN ('setup', 'reset')),
    code_hash  BYTEA NOT NULL,
    attempts   INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX admin_codes_lookup_idx ON admin_codes (email, purpose, expires_at);

-- Every admin action (sign-in, catalog edits, password changes) — the panel's
-- "Журнал действий" tab. admin_id is nullable for pre-auth events (failed
-- logins). details never carries message content or user PII.
CREATE TABLE admin_audit_log (
    id         BIGSERIAL PRIMARY KEY,
    admin_id   UUID,
    action     TEXT NOT NULL,
    entity     TEXT,
    entity_id  TEXT,
    details    JSONB,
    ip         INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX admin_audit_log_created_idx ON admin_audit_log (created_at DESC);

-- Server error feed for the panel's "Ошибки" tab: every 5xx response (panics
-- included) lands here with route pattern + request_id, never request bodies
-- or PII (CLAUDE.md invariant #3). Old rows are purged by cmd/cleanup.
CREATE TABLE admin_error_events (
    id         BIGSERIAL PRIMARY KEY,
    source     TEXT NOT NULL,
    route      TEXT NOT NULL,
    method     TEXT NOT NULL,
    status     INT NOT NULL,
    request_id TEXT,
    message    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX admin_error_events_created_idx ON admin_error_events (created_at DESC);

-- Optional retail price in whole rubles, set from the admin panel. The model
-- mentions it only when the user explicitly asks (prompt rule), so the catalog
-- recommendation stays native.
ALTER TABLE fertilizers ADD COLUMN price_rub INT;

-- +goose Down
ALTER TABLE fertilizers DROP COLUMN IF EXISTS price_rub;
DROP TABLE IF EXISTS admin_error_events;
DROP TABLE IF EXISTS admin_audit_log;
DROP TABLE IF EXISTS admin_codes;
DROP TABLE IF EXISTS admin_sessions;
DROP TABLE IF EXISTS admin_users;
