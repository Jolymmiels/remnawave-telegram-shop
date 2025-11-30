-- Add recurring payment fields to customer table
ALTER TABLE customer ADD COLUMN IF NOT EXISTS payment_method_id VARCHAR(255);
ALTER TABLE customer ADD COLUMN IF NOT EXISTS autopay_enabled BOOLEAN DEFAULT true;
ALTER TABLE customer ADD COLUMN IF NOT EXISTS autopay_plan_id BIGINT REFERENCES plan(id);
ALTER TABLE customer ADD COLUMN IF NOT EXISTS autopay_months INTEGER DEFAULT 1;

-- Add recurring payment settings
INSERT INTO settings (key, value, updated_at) VALUES 
    ('recurring_payments_enabled', 'false', NOW()),
    ('recurring_days_before', '1', NOW()),
    ('recurring_notify_days_before', '3', NOW())
ON CONFLICT (key) DO NOTHING;
