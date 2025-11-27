-- +goose Up

-- Добавляем plan_id в purchase
ALTER TABLE purchase ADD COLUMN plan_id BIGINT REFERENCES plan(id) ON DELETE SET NULL;

-- Привязываем существующие покупки к дефолтному тарифу
UPDATE purchase SET plan_id = (SELECT id FROM plan WHERE is_default = TRUE LIMIT 1)
WHERE plan_id IS NULL;

-- Индекс для поиска покупок по тарифу
CREATE INDEX idx_purchase_plan_id ON purchase(plan_id);
