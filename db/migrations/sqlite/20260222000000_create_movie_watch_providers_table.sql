-- migrate:up
CREATE TABLE IF NOT EXISTS movie_watch_providers (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
	provider_name TEXT NOT NULL,
	provider_id INTEGER NOT NULL,
	logo_path TEXT,
	provider_type TEXT NOT NULL CHECK(provider_type IN ('stream', 'rent', 'buy')),
	display_priority INTEGER DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_mwp_movie_id ON movie_watch_providers(movie_id);
CREATE UNIQUE INDEX idx_mwp_movie_provider ON movie_watch_providers(movie_id, provider_id, provider_type);

-- migrate:down
DROP INDEX IF EXISTS idx_mwp_movie_provider;
DROP INDEX IF EXISTS idx_mwp_movie_id;
DROP TABLE IF EXISTS movie_watch_providers;
