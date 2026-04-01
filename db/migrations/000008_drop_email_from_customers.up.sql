DROP INDEX IF EXISTS idx_customer_email_unique;

ALTER TABLE customer
    DROP COLUMN IF EXISTS email;
