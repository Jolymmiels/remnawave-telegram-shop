-- +goose Up

CREATE TABLE IF NOT EXISTS plan (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price_1 INTEGER NOT NULL DEFAULT 0,
    price_3 INTEGER NOT NULL DEFAULT 0,
    price_6 INTEGER NOT NULL DEFAULT 0,
    price_12 INTEGER NOT NULL DEFAULT 0,
    traffic_limit INTEGER NOT NULL DEFAULT 0,
    device_limit INTEGER,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Создаем базовый тариф из текущих настроек
INSERT INTO plan (name, price_1, price_3, price_6, price_12, traffic_limit, is_default, is_active)
SELECT 
    'Базовый',
    COALESCE((SELECT value::int FROM settings WHERE key = 'price_1'), 1),
    COALESCE((SELECT value::int FROM settings WHERE key = 'price_3'), 1),
    COALESCE((SELECT value::int FROM settings WHERE key = 'price_6'), 1),
    COALESCE((SELECT value::int FROM settings WHERE key = 'price_12'), 1),
    COALESCE((SELECT value::int FROM settings WHERE key = 'traffic_limit'), 0),
    TRUE, TRUE
WHERE NOT EXISTS (SELECT 1 FROM plan WHERE is_default = TRUE);

-- Добавляем курс конвертации в settings
INSERT INTO settings (key, value) VALUES ('stars_exchange_rate', '1.5')
ON CONFLICT (key) DO NOTHING;

-- Удаляем устаревшие настройки цен (они теперь в тарифах)
DELETE FROM settings WHERE key IN (
    'price_1', 'price_3', 'price_6', 'price_12',
    'stars_price_1', 'stars_price_3', 'stars_price_6', 'stars_price_12'
);

-- Уникальный индекс для is_default (только один дефолтный тариф)
CREATE UNIQUE INDEX idx_plan_default ON plan (is_default) WHERE is_default = TRUE;

CREATE INDEX idx_plan_active ON plan (is_active);
