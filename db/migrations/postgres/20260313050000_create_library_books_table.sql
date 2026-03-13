-- migrate:up
CREATE TABLE IF NOT EXISTS library_books (
	library_id TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
	book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	PRIMARY KEY (library_id, book_id)
);

CREATE INDEX idx_library_books_book_id ON library_books(book_id);

-- migrate:down
DROP TABLE IF EXISTS library_books;
