-- migrate:up
-- SQLite does not support ALTER TABLE ... ADD CONSTRAINT, so we recreate the table
-- with the foreign key constraint on user_id.
CREATE TABLE passkey_challenges_new (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
	session_data TEXT NOT NULL,
	expires_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO passkey_challenges_new (id, user_id, session_data, expires_at, created_at)
SELECT id, user_id, session_data, expires_at, created_at FROM passkey_challenges;

DROP TABLE passkey_challenges;
ALTER TABLE passkey_challenges_new RENAME TO passkey_challenges;

CREATE INDEX IF NOT EXISTS idx_passkey_challenges_expires_at ON passkey_challenges (expires_at);

-- migrate:down
CREATE TABLE passkey_challenges_old (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT,
	session_data TEXT NOT NULL,
	expires_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO passkey_challenges_old (id, user_id, session_data, expires_at, created_at)
SELECT id, user_id, session_data, expires_at, created_at FROM passkey_challenges;

DROP TABLE passkey_challenges;
ALTER TABLE passkey_challenges_old RENAME TO passkey_challenges;

CREATE INDEX IF NOT EXISTS idx_passkey_challenges_expires_at ON passkey_challenges (expires_at);
