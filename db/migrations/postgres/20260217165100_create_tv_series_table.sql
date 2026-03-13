-- migrate:up
CREATE TABLE IF NOT EXISTS tv_series (
	id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
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
	first_aired TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tv_series_tmdb_id ON tv_series(tmdb_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tv_series_imdb_id ON tv_series(imdb_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tv_series_tvdb_id ON tv_series(tvdb_id);

CREATE TABLE IF NOT EXISTS user_tv_series (
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	tv_series_id TEXT NOT NULL REFERENCES tv_series(id) ON DELETE CASCADE,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	PRIMARY KEY (user_id, tv_series_id)
);

CREATE INDEX IF NOT EXISTS idx_user_tv_series_tv_series_id ON user_tv_series(tv_series_id);

-- migrate:down
DROP INDEX IF EXISTS idx_user_tv_series_tv_series_id;
DROP TABLE IF EXISTS user_tv_series;
DROP INDEX IF EXISTS idx_tv_series_tvdb_id;
DROP INDEX IF EXISTS idx_tv_series_imdb_id;
DROP INDEX IF EXISTS idx_tv_series_tmdb_id;
DROP TABLE IF EXISTS tv_series;
