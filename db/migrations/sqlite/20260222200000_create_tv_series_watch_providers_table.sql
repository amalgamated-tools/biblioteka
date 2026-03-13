-- migrate:up
CREATE TABLE IF NOT EXISTS tv_series_watch_providers (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	tv_series_id TEXT NOT NULL REFERENCES tv_series(id) ON DELETE CASCADE,
	provider_name TEXT NOT NULL,
	provider_id INTEGER NOT NULL,
	logo_path TEXT,
	provider_type TEXT NOT NULL CHECK(provider_type IN ('stream', 'buy')),
	display_priority INTEGER DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_tswp_tv_series_id ON tv_series_watch_providers(tv_series_id);
CREATE UNIQUE INDEX idx_tswp_series_provider ON tv_series_watch_providers(tv_series_id, provider_id, provider_type);

ALTER TABLE tv_series ADD COLUMN providers_fetched_at DATETIME;

-- migrate:down
ALTER TABLE tv_series DROP COLUMN providers_fetched_at;
DROP INDEX IF EXISTS idx_tswp_series_provider;
DROP INDEX IF EXISTS idx_tswp_tv_series_id;
DROP TABLE IF EXISTS tv_series_watch_providers;
