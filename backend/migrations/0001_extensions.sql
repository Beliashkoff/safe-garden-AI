-- +goose Up
-- citext powers case-insensitive email storage (users.email, email_codes.email).
-- pgcrypto provides gen_random_uuid() used as the default for all UUID columns.
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- +goose Down
-- We do not drop extensions in down: other schemas in the same database may
-- depend on them. Removing extensions is a manual ops decision, not a
-- migration concern.
SELECT 1;
