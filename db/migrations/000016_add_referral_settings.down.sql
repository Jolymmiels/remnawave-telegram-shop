DELETE FROM settings WHERE key IN (
    'referral_enabled',
    'referral_bonus_days',
    'referral_referee_bonus_days',
    'referral_tiers_enabled',
    'referral_tier1_threshold',
    'referral_tier1_bonus',
    'referral_tier2_threshold',
    'referral_tier2_bonus',
    'referral_tier3_threshold',
    'referral_tier3_bonus',
    'referral_recurring_enabled',
    'referral_recurring_percent'
);
