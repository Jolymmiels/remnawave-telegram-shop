DROP INDEX IF EXISTS idx_customer_is_blocked_by_user;
ALTER TABLE customer DROP COLUMN IF EXISTS is_blocked_by_user;
