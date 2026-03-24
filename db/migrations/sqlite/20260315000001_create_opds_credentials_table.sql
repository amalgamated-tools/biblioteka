-- migrate:up
CREATE TABLE IF NOT EXISTS opds_credentials (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    username TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_opds_credentials_username ON opds_credentials (LOWER(username));
CREATE INDEX IF NOT EXISTS idx_opds_credentials_user_id ON opds_credentials (user_id);

-- migrate:down
DROP TABLE IF EXISTS opds_credentials;
