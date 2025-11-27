-- +goose Up

ALTER TABLE broadcast
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS total_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sent_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS failed_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS blocked_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE broadcast
    ADD CONSTRAINT broadcast_status_chk
    CHECK (status IN ('pending', 'in_progress', 'completed', 'failed'));

CREATE INDEX IF NOT EXISTS idx_broadcast_status ON broadcast (status);
