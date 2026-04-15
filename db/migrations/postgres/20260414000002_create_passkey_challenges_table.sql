-- migrate:up
CREATE TABLE IF NOT EXISTS passkey_challenges (
	id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	user_id TEXT,
	session_data TEXT NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_passkey_challenges_expires_at ON passkey_challenges (expires_at);

-- migrate:down
DROP TABLE IF EXISTS passkey_challenges;
