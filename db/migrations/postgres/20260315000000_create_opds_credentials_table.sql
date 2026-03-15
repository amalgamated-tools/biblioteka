-- migrate:up
CREATE TABLE IF NOT EXISTS opds_credentials (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_opds_credentials_username ON opds_credentials (username);
CREATE INDEX IF NOT EXISTS idx_opds_credentials_user_id ON opds_credentials (user_id);

-- migrate:down
DROP TABLE IF EXISTS opds_credentials;
