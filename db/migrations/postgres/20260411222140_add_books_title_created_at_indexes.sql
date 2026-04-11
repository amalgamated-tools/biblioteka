-- migrate:up
-- Index to speed up ORDER BY title ASC used in ListBooksPaginated,
-- ListBooksByAuthorPaginated, ListBooksBySeriesPaginated, and SearchBooks.
CREATE INDEX IF NOT EXISTS idx_books_title ON books (title);

-- Index to speed up ORDER BY created_at DESC used in ListRecentBooks.
CREATE INDEX IF NOT EXISTS idx_books_created_at ON books (created_at DESC);

-- migrate:down
DROP INDEX IF EXISTS idx_books_title;
DROP INDEX IF EXISTS idx_books_created_at;
