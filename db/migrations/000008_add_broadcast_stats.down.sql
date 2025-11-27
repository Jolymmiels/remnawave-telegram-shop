-- +goose Down

ALTER TABLE broadcast DROP CONSTRAINT IF EXISTS broadcast_status_chk;

DROP INDEX IF EXISTS idx_broadcast_status;

ALTER TABLE broadcast
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS total_count,
    DROP COLUMN IF EXISTS sent_count,
    DROP COLUMN IF EXISTS failed_count,
    DROP COLUMN IF EXISTS blocked_count;
