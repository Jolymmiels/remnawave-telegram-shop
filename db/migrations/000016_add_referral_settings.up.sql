-- Referral program settings
INSERT INTO settings (key, value) VALUES
    ('referral_enabled', 'true'),
    ('referral_bonus_days', '3'),
    ('referral_referee_bonus_days', '0'),
    ('referral_tiers_enabled', 'false'),
    ('referral_tier1_threshold', '5'),
    ('referral_tier1_bonus', '3'),
    ('referral_tier2_threshold', '15'),
    ('referral_tier2_bonus', '5'),
    ('referral_tier3_threshold', '30'),
    ('referral_tier3_bonus', '7'),
    ('referral_recurring_enabled', 'false'),
    ('referral_recurring_percent', '10')
ON CONFLICT (key) DO NOTHING;
