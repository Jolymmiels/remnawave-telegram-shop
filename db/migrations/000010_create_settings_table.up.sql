-- +goose Up

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default values from environment (will be populated on first run)
INSERT INTO settings (key, value) VALUES
    ('price_1', '1'),
    ('price_3', '1'),
    ('price_6', '1'),
    ('price_12', '1'),
    ('stars_price_1', '1'),
    ('stars_price_3', '321'),
    ('stars_price_6', '674'),
    ('stars_price_12', '31231'),
    ('referral_days', '7'),
    ('mini_app_url', ''),
    ('trial_traffic_limit', '20'),
    ('trial_days', '3'),
    ('trial_remnawave_tag', ''),
    ('crypto_pay_enabled', 'false'),
    ('crypto_pay_token', ''),
    ('crypto_pay_url', 'https://pay.crypt.bot'),
    ('yookasa_enabled', 'false'),
    ('yookasa_secret_key', ''),
    ('yookasa_shop_id', ''),
    ('yookasa_url', 'https://api.yookassa.ru/v3'),
    ('yookasa_email', ''),
    ('telegram_stars_enabled', 'false'),
    ('traffic_limit', '0'),
    ('server_status_url', ''),
    ('support_url', ''),
    ('feedback_url', ''),
    ('channel_url', ''),
    ('tribute_webhook_url', ''),
    ('tribute_api_key', ''),
    ('tribute_payment_url', ''),
    ('is_web_app_link', 'false'),
    ('days_in_month', '31'),
    ('remnawave_tag', ''),
    ('blocked_telegram_ids', ''),
    ('require_paid_purchase_for_stars', 'false'),
    ('squad_uuids', ''),
    ('external_squad_uuid', ''),
    ('trial_internal_squads', ''),
    ('trial_external_squad_uuid', '')
ON CONFLICT (key) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_settings_key ON settings (key);
