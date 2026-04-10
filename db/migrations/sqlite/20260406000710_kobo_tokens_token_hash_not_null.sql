-- migrate:up
-- SQLite does not support ALTER COLUMN, so rebuild the table to add NOT NULL on token_hash.
CREATE TABLE kobo_tokens_new (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	token_hash TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO kobo_tokens_new (id, user_id, name, token_hash, created_at)
SELECT id, user_id, name, token_hash, created_at FROM kobo_tokens;

DROP TABLE kobo_tokens;
ALTER TABLE kobo_tokens_new RENAME TO kobo_tokens;

CREATE UNIQUE INDEX IF NOT EXISTS idx_kobo_tokens_token_hash ON kobo_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_kobo_tokens_user_id ON kobo_tokens (user_id);

-- migrate:down
-- Reverting to nullable token_hash requires another table rebuild.
CREATE TABLE kobo_tokens_old (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	token_hash TEXT,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO kobo_tokens_old (id, user_id, name, token_hash, created_at)
SELECT id, user_id, name, token_hash, created_at FROM kobo_tokens;

DROP TABLE kobo_tokens;
ALTER TABLE kobo_tokens_old RENAME TO kobo_tokens;

CREATE UNIQUE INDEX IF NOT EXISTS idx_kobo_tokens_token_hash ON kobo_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_kobo_tokens_user_id ON kobo_tokens (user_id);
