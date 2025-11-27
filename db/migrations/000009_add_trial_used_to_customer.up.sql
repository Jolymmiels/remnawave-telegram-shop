ALTER TABLE customer ADD COLUMN trial_used BOOLEAN NOT NULL DEFAULT FALSE;

-- Set trial_used = true for users who already have/had a subscription
UPDATE customer SET trial_used = TRUE WHERE subscription_link IS NOT NULL;
