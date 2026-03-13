-- migrate:up
CREATE TABLE media_services (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	name TEXT NOT NULL,
	type TEXT NOT NULL CHECK (type IN ('radarr', 'sonarr', 'prowlarr', 'seerr')),
	url TEXT NOT NULL,
	api_key TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO media_services (id, name, type, url, api_key, created_at, updated_at)
	SELECT id, name, type, url, api_key, created_at, updated_at
	FROM arr_services;

DROP TABLE arr_services;

-- migrate:down
ALTER TABLE media_services RENAME TO arr_services;
ALTER TABLE arr_services ADD COLUMN user_id TEXT REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_arr_services_user_id ON arr_services(user_id);
