-- Remove recurring payment fields from customer table
ALTER TABLE customer DROP COLUMN IF EXISTS payment_method_id;
ALTER TABLE customer DROP COLUMN IF EXISTS autopay_enabled;
ALTER TABLE customer DROP COLUMN IF EXISTS autopay_plan_id;
ALTER TABLE customer DROP COLUMN IF EXISTS autopay_months;

-- Remove recurring payment settings
DELETE FROM settings WHERE key IN ('recurring_payments_enabled', 'recurring_days_before', 'recurring_notify_days_before');
