-- migrate:up
CREATE TABLE IF NOT EXISTS goodreads_metadata (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	book_id TEXT REFERENCES books(id) ON DELETE SET NULL,
	status TEXT NOT NULL DEFAULT 'pending',
	title TEXT,
	description TEXT,
	asin TEXT,
	isbn10 TEXT,
	isbn13 TEXT,
	goodreads_id TEXT,
	publication_date TEXT,
	publisher TEXT,
	language TEXT,
	cover_image_url TEXT,
	author_name TEXT,
	author_goodreads_id TEXT,
	author_image_url TEXT,
	goodreads_work_id TEXT,
	goodreads_book_legacy_id INTEGER,
	goodreads_work_legacy_id INTEGER,
	goodreads_author_legacy_id INTEGER,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_goodreads_metadata_user_id
	ON goodreads_metadata (user_id);

CREATE INDEX IF NOT EXISTS idx_goodreads_metadata_user_status_created_at_id
	ON goodreads_metadata (user_id, status, created_at DESC, id DESC);

-- migrate:down
DROP INDEX IF EXISTS idx_goodreads_metadata_user_status_created_at_id;
DROP INDEX IF EXISTS idx_goodreads_metadata_user_id;
DROP TABLE IF EXISTS goodreads_metadata;
