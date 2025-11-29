INSERT INTO settings (key, value) VALUES ('trial_device_limit', '3') ON CONFLICT (key) DO NOTHING;
