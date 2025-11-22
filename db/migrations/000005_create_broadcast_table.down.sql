-- +goose Down
DROP INDEX IF EXISTS idx_broadcast_language_created_at;
DROP INDEX IF EXISTS idx_broadcast_type_created_at;
DROP INDEX IF EXISTS idx_broadcast_created_at;

DROP TABLE IF EXISTS broadcast;
