ALTER TABLE customer
    ADD COLUMN IF NOT EXISTS email TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_email_unique
    ON customer (email)
    WHERE email IS NOT NULL;
