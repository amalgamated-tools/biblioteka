-- migrate:up
CREATE TABLE IF NOT EXISTS api_keys (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	key_hash TEXT NOT NULL,
	key_prefix TEXT NOT NULL,
	last_used_at DATETIME,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys (key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys (user_id);

-- migrate:down
DROP TABLE IF EXISTS api_keys;
