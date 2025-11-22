DROP INDEX IF EXISTS idx_customer_is_blocked;
ALTER TABLE customer DROP COLUMN is_blocked;
