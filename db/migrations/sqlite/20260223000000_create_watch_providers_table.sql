-- migrate:up
CREATE TABLE IF NOT EXISTS watch_providers (
	provider_id INTEGER PRIMARY KEY,
	provider_name TEXT NOT NULL,
	logo_path TEXT,
	display_priority INTEGER DEFAULT 0,
	provider_type TEXT NOT NULL DEFAULT 'both' CHECK(provider_type IN ('movie', 'tv', 'both')),
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_watch_providers_display_priority_name
	ON watch_providers (display_priority, provider_name);

-- migrate:down
DROP INDEX IF EXISTS idx_watch_providers_display_priority_name;
DROP TABLE IF EXISTS watch_providers;
