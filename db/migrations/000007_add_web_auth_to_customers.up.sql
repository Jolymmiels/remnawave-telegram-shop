ALTER TABLE customer
    ADD COLUMN login TEXT,
    ADD COLUMN password_hash TEXT,
    ADD COLUMN auth_type VARCHAR(20) NOT NULL DEFAULT 'telegram',
    ADD COLUMN remnawave_user_uuid UUID,
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN last_login_at TIMESTAMP WITH TIME ZONE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_login_unique
    ON customer (login)
    WHERE login IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_remnawave_user_uuid_unique
    ON customer (remnawave_user_uuid)
    WHERE remnawave_user_uuid IS NOT NULL;
