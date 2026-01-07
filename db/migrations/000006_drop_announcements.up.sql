-- Drop indexes
DROP INDEX IF EXISTS idx_announcement_delivery_customer_id;
DROP INDEX IF EXISTS idx_announcement_delivery_announcement_id;

-- Drop constraint
ALTER TABLE IF EXISTS announcement_delivery
    DROP CONSTRAINT IF EXISTS uq_announcement_delivery;

-- Drop announcement_delivery table
DROP TABLE IF EXISTS announcement_delivery;

-- Drop announcement indexes
DROP INDEX IF EXISTS idx_announcement_created_at;
DROP INDEX IF EXISTS idx_announcement_is_active;
DROP INDEX IF EXISTS idx_announcement_expires_at;

-- Drop announcement table
DROP TABLE IF EXISTS announcement;
