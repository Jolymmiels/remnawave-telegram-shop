DROP INDEX IF EXISTS idx_customer_remnawave_user_uuid_unique;
DROP INDEX IF EXISTS idx_customer_login_unique;

ALTER TABLE customer
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS remnawave_user_uuid,
    DROP COLUMN IF EXISTS auth_type,
    DROP COLUMN IF EXISTS password_hash,
    DROP COLUMN IF EXISTS login;
