DROP INDEX IF EXISTS idx_customer_merged_into_customer_id;

ALTER TABLE customer
    DROP COLUMN IF EXISTS merged_into_customer_id;
