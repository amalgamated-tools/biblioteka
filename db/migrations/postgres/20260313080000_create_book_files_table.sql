-- migrate:up
CREATE TABLE IF NOT EXISTS book_files (
	id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
	file_type TEXT NOT NULL,
	file_name TEXT NOT NULL,
	file_size INTEGER NOT NULL,
	file_hash TEXT,
	file_path TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_book_files_book_id ON book_files(book_id);

-- migrate:down
DROP TABLE IF EXISTS book_files;
