ALTER TABLE customer ADD COLUMN is_blocked BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_customer_is_blocked ON customer(is_blocked);
