-- migrate:up
CREATE TABLE IF NOT EXISTS passkey_challenges (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT,
	session_data TEXT NOT NULL,
	expires_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_passkey_challenges_expires_at ON passkey_challenges (expires_at);

-- migrate:down
DROP TABLE IF EXISTS passkey_challenges;
