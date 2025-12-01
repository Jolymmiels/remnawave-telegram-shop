-- +goose Down
-- Revert broadcast type constraint

ALTER TABLE broadcast DROP CONSTRAINT IF EXISTS broadcast_type_chk;

ALTER TABLE broadcast ADD CONSTRAINT broadcast_type_chk
    CHECK (type IN ('all', 'active', 'inactive'));
