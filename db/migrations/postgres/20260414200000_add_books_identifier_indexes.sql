-- migrate:up
CREATE INDEX IF NOT EXISTS idx_books_isbn13 ON books (isbn13) WHERE isbn13 IS NOT NULL AND isbn13 != '';
CREATE INDEX IF NOT EXISTS idx_books_isbn10 ON books (isbn10) WHERE isbn10 IS NOT NULL AND isbn10 != '';
CREATE INDEX IF NOT EXISTS idx_books_asin ON books (asin) WHERE asin IS NOT NULL AND asin != '';
CREATE INDEX IF NOT EXISTS idx_books_goodreads_id ON books (goodreads_id) WHERE goodreads_id IS NOT NULL AND goodreads_id != '';

-- migrate:down
DROP INDEX IF EXISTS idx_books_isbn13;
DROP INDEX IF EXISTS idx_books_isbn10;
DROP INDEX IF EXISTS idx_books_asin;
DROP INDEX IF EXISTS idx_books_goodreads_id;
