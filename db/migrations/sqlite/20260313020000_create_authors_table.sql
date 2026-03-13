-- migrate:up
CREATE TABLE IF NOT EXISTS authors (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	name TEXT NOT NULL UNIQUE,
	goodreads_id TEXT,
	hardcover_id TEXT,
	google_books_id TEXT,
	image_url TEXT,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- migrate:down
DROP TABLE IF EXISTS authors;
