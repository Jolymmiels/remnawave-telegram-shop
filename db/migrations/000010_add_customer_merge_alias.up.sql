ALTER TABLE customer
    ADD COLUMN merged_into_customer_id BIGINT REFERENCES customer (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_customer_merged_into_customer_id
    ON customer (merged_into_customer_id);
