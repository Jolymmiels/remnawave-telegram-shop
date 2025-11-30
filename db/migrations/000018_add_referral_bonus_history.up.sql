CREATE TABLE IF NOT EXISTS referral_bonus_history (
    id             BIGSERIAL PRIMARY KEY,
    referral_id    BIGINT NOT NULL REFERENCES referral(id) ON DELETE CASCADE,
    purchase_id    BIGINT REFERENCES purchase(id) ON DELETE SET NULL,
    bonus_days     INT NOT NULL,
    is_first_bonus BOOLEAN NOT NULL DEFAULT FALSE,
    granted_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_referral_bonus_history_referral_id ON referral_bonus_history(referral_id);
CREATE INDEX IF NOT EXISTS idx_referral_bonus_history_granted_at ON referral_bonus_history(granted_at DESC);
