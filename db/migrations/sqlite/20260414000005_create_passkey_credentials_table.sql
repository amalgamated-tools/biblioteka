-- migrate:up
CREATE TABLE IF NOT EXISTS passkey_credentials (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	credential_id TEXT NOT NULL,
	credential_data TEXT NOT NULL,
	aaguid TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_passkey_credentials_credential_id ON passkey_credentials (credential_id);
CREATE INDEX IF NOT EXISTS idx_passkey_credentials_user_id ON passkey_credentials (user_id);

-- migrate:down
DROP TABLE IF EXISTS passkey_credentials;
