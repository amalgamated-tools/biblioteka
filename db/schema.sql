CREATE TABLE schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
CREATE TABLE users (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE COLLATE NOCASE,
	password_hash TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
, oidc_subject TEXT, is_admin INTEGER NOT NULL DEFAULT 0);
CREATE TABLE movies (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	title TEXT NOT NULL,
	sort_title TEXT,
	original_title TEXT,
	overview TEXT,
	year INTEGER,
	runtime INTEGER,
	certification TEXT,
	studio TEXT,
	genres TEXT,
	status TEXT NOT NULL DEFAULT 'unknown',
	imdb_id TEXT,
	tmdb_id INTEGER,
	youtube_trailer_id TEXT,
	poster_url TEXT,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
, providers_fetched_at DATETIME);
CREATE UNIQUE INDEX idx_movies_tmdb_id ON movies(tmdb_id);
CREATE UNIQUE INDEX idx_movies_imdb_id ON movies(imdb_id);
CREATE TABLE user_movies (
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	PRIMARY KEY (user_id, movie_id)
);
CREATE INDEX idx_user_movies_movie_id ON user_movies(movie_id);
CREATE TABLE tv_series (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	title TEXT NOT NULL,
	sort_title TEXT,
	overview TEXT,
	year INTEGER,
	runtime INTEGER,
	certification TEXT,
	network TEXT,
	genres TEXT,
	status TEXT NOT NULL DEFAULT 'unknown',
	series_type TEXT NOT NULL DEFAULT 'standard',
	imdb_id TEXT,
	tmdb_id INTEGER,
	tvdb_id INTEGER,
	poster_url TEXT,
	first_aired DATETIME,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
, providers_fetched_at DATETIME);
CREATE UNIQUE INDEX idx_tv_series_tmdb_id ON tv_series(tmdb_id);
CREATE UNIQUE INDEX idx_tv_series_imdb_id ON tv_series(imdb_id);
CREATE UNIQUE INDEX idx_tv_series_tvdb_id ON tv_series(tvdb_id);
CREATE TABLE user_tv_series (
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	tv_series_id TEXT NOT NULL REFERENCES tv_series(id) ON DELETE CASCADE,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	PRIMARY KEY (user_id, tv_series_id)
);
CREATE INDEX idx_user_tv_series_tv_series_id ON user_tv_series(tv_series_id);
CREATE TABLE movie_watch_providers (
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
CREATE TABLE settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE tv_series_watch_providers (
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
CREATE UNIQUE INDEX idx_users_oidc_subject ON users(oidc_subject) WHERE oidc_subject IS NOT NULL;
CREATE TABLE watch_providers (
	provider_id INTEGER PRIMARY KEY,
	provider_name TEXT NOT NULL,
	logo_path TEXT,
	display_priority INTEGER DEFAULT 0,
	provider_type TEXT NOT NULL DEFAULT 'both' CHECK(provider_type IN ('movie', 'tv', 'both')),
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_watch_providers_display_priority_name
	ON watch_providers (display_priority, provider_name);
CREATE TABLE media_services (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	name TEXT NOT NULL,
	type TEXT NOT NULL CHECK (type IN ('radarr', 'sonarr', 'prowlarr', 'seerr')),
	url TEXT NOT NULL,
	api_key TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE user_watch_providers (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id INTEGER NOT NULL REFERENCES watch_providers(provider_id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, provider_id)
);
CREATE INDEX idx_user_watch_providers_provider_id ON user_watch_providers(provider_id);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20260214235631_create_users_table'),
  ('20260216120000_create_arr_services_table'),
  ('20260217164400_create_movies_table'),
  ('20260217165100_create_tv_series_table'),
  ('20260221000000_create_goqite_table'),
  ('20260222000000_create_movie_watch_providers_table'),
  ('20260222100000_add_providers_fetched_at_to_movies'),
  ('20260222200000_create_settings_table'),
  ('20260222200000_create_tv_series_watch_providers_table'),
  ('20260222210000_add_oidc_to_users'),
  ('20260223000000_create_watch_providers_table'),
  ('20260224000000_add_is_admin_to_users'),
  ('20260225000000_make_arr_services_instance_wide'),
  ('20260226000000_create_user_watch_providers_table'),
  ('20260226000000_drop_goqite_table');
