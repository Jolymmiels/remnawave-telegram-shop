ALTER TABLE customer ADD COLUMN remnawave_id BIGINT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_remnawave_id
    ON customer (remnawave_id)
    WHERE remnawave_id IS NOT NULL;
