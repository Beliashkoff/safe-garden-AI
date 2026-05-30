-- +goose Up
-- Tracks whether the user's Object Storage media (prefix u/{user_id}/) has been
-- purged after account deletion. A deleted-but-unpurged user
-- (deleted_at IS NOT NULL AND media_purged_at IS NULL) is the work-list for the
-- cleanup job; setting media_purged_at marks it done. This makes the async purge
-- durable: a crash mid-run leaves the marker NULL, so the next run retries.
ALTER TABLE users ADD COLUMN media_purged_at TIMESTAMPTZ;

-- Partial index for the cleanup job's pending-purge scan (small, only unpurged).
CREATE INDEX users_pending_media_purge_idx
    ON users (deleted_at)
    WHERE media_purged_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS users_pending_media_purge_idx;
ALTER TABLE users DROP COLUMN IF EXISTS media_purged_at;
