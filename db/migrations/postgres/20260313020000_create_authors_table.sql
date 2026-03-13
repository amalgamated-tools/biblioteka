-- migrate:up
CREATE TABLE IF NOT EXISTS authors (
	id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	name TEXT NOT NULL UNIQUE,
	goodreads_id TEXT,
	hardcover_id TEXT,
	google_books_id TEXT,
	image_url TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- migrate:down
DROP TABLE IF EXISTS authors;
