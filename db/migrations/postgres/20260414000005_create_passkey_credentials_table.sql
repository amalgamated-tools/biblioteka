-- migrate:up
CREATE TABLE IF NOT EXISTS passkey_credentials (
	id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	credential_id TEXT NOT NULL,
	credential_data TEXT NOT NULL,
	aaguid TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_passkey_credentials_credential_id ON passkey_credentials (credential_id);
CREATE INDEX IF NOT EXISTS idx_passkey_credentials_user_id ON passkey_credentials (user_id);

-- migrate:down
DROP TABLE IF EXISTS passkey_credentials;
