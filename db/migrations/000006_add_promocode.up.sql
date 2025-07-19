CREATE TABLE IF NOT EXISTS promocode (
    id SERIAL PRIMARY KEY,
    code VARCHAR(64) UNIQUE NOT NULL,
    months INT NOT NULL,
    uses_left INT NOT NULL,
    created_by BIGINT NOT NULL REFERENCES customer(telegram_id),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
