-- migrate:up
CREATE TABLE user_watch_providers (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id INTEGER NOT NULL REFERENCES watch_providers(provider_id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, provider_id)
);

CREATE INDEX idx_user_watch_providers_provider_id ON user_watch_providers(provider_id);

-- migrate:down
DROP INDEX IF EXISTS idx_user_watch_providers_provider_id;
DROP TABLE IF EXISTS user_watch_providers;
