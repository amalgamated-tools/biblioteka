-- migrate:up
CREATE TABLE IF NOT EXISTS movies (
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
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_movies_tmdb_id ON movies(tmdb_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_movies_imdb_id ON movies(imdb_id);

CREATE TABLE IF NOT EXISTS user_movies (
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	PRIMARY KEY (user_id, movie_id)
);

CREATE INDEX IF NOT EXISTS idx_user_movies_movie_id ON user_movies(movie_id);

-- migrate:down
DROP INDEX IF EXISTS idx_user_movies_movie_id;
DROP TABLE IF EXISTS user_movies;
DROP INDEX IF EXISTS idx_movies_imdb_id;
DROP INDEX IF EXISTS idx_movies_tmdb_id;
DROP TABLE IF EXISTS movies;
