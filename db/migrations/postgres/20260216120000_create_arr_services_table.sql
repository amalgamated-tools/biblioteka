-- migrate:up
CREATE TABLE IF NOT EXISTS arr_services (
	id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	type TEXT NOT NULL CHECK (type IN ('radarr', 'sonarr', 'prowlarr', 'seerr')),
	url TEXT NOT NULL,
	api_key TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_arr_services_user_id ON arr_services(user_id);

-- migrate:down
DROP INDEX IF EXISTS idx_arr_services_user_id;
DROP TABLE IF EXISTS arr_services;
