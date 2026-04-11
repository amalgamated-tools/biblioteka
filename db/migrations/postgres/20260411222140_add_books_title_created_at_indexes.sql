-- migrate:up
-- Index to speed up ORDER BY title ASC, id ASC used in ListBooksPaginated,
-- ListBooksByAuthorPaginated, ListBooksBySeriesPaginated, and SearchBooks.
CREATE INDEX IF NOT EXISTS idx_books_title ON books (title, id);

-- Index to speed up ORDER BY created_at DESC, id DESC used in ListRecentBooks.
CREATE INDEX IF NOT EXISTS idx_books_created_at ON books (created_at DESC, id DESC);

-- migrate:down
DROP INDEX IF EXISTS idx_books_title;
DROP INDEX IF EXISTS idx_books_created_at;
