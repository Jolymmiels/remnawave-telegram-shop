CREATE TABLE promo (
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(50) NOT NULL UNIQUE,
    bonus_days INTEGER NOT NULL, -- Additional days to add to subscription
    max_uses   INTEGER DEFAULT NULL, -- NULL for unlimited uses
    used_count INTEGER DEFAULT 0,
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT NULL, -- NULL for no expiration
    active     BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_promo_code ON promo(code);
CREATE INDEX idx_promo_active ON promo(active);

-- Track promo code usage by customers
CREATE TABLE promo_usage (
    id          BIGSERIAL PRIMARY KEY,
    promo_id    BIGINT NOT NULL REFERENCES promo(id) ON DELETE CASCADE,
    customer_id BIGINT NOT NULL REFERENCES customer(id) ON DELETE CASCADE,
    used_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(promo_id, customer_id) -- Prevent duplicate usage by same customer
);