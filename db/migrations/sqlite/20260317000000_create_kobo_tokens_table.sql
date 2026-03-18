-- migrate:up
CREATE TABLE IF NOT EXISTS kobo_tokens (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	token TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_kobo_tokens_token ON kobo_tokens (token);
CREATE INDEX IF NOT EXISTS idx_kobo_tokens_user_id ON kobo_tokens (user_id);

-- migrate:down
DROP TABLE IF EXISTS kobo_tokens;
