ALTER TABLE customer ADD COLUMN is_blocked_by_user BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_customer_is_blocked_by_user ON customer(is_blocked_by_user);
