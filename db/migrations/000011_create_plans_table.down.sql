-- +goose Down

DROP INDEX IF EXISTS idx_plan_default;
DROP INDEX IF EXISTS idx_plan_active;
DROP TABLE IF EXISTS plan;

-- Восстанавливаем настройки цен
INSERT INTO settings (key, value) VALUES
    ('price_1', '1'),
    ('price_3', '1'),
    ('price_6', '1'),
    ('price_12', '1'),
    ('stars_price_1', '1'),
    ('stars_price_3', '1'),
    ('stars_price_6', '1'),
    ('stars_price_12', '1')
ON CONFLICT (key) DO NOTHING;

DELETE FROM settings WHERE key = 'stars_exchange_rate';
