-- migrate:up
CREATE TABLE IF NOT EXISTS book_authors (
	book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
	author_id TEXT NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	PRIMARY KEY (book_id, author_id)
);

CREATE INDEX idx_book_authors_author_id ON book_authors(author_id);

-- migrate:down
DROP TABLE IF EXISTS book_authors;
