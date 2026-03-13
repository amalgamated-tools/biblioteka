-- migrate:up
CREATE TABLE IF NOT EXISTS books (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	title TEXT NOT NULL,
	description TEXT,
	asin TEXT,
	isbn10 TEXT,
	isbn13 TEXT,
	goodreads_id TEXT,
	hardcover_id TEXT,
	google_books_id TEXT,
	publication_date TEXT,
	publisher TEXT,
	language TEXT,
	num_pages INTEGER,
	cover_image_url TEXT,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- migrate:down
DROP TABLE IF EXISTS books;
