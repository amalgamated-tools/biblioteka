-- migrate:up
INSERT INTO settings (key, value, updated_at)
VALUES ('registration_disabled', 'false', datetime('now'))
ON CONFLICT (key) DO NOTHING;

-- migrate:down
DELETE FROM settings WHERE key = 'registration_disabled';
