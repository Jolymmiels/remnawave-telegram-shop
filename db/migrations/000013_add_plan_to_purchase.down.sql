-- +goose Down

DROP INDEX IF EXISTS idx_purchase_plan_id;
ALTER TABLE purchase DROP COLUMN IF EXISTS plan_id;
