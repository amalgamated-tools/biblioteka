-- migrate:up
-- Extend idx_books_created_at to include the id tiebreaker column so
-- ListRecentBooks (ORDER BY created_at DESC, id DESC) is fully covered
-- by the index and eliminates the sort.
DROP INDEX IF EXISTS idx_books_created_at;
CREATE INDEX IF NOT EXISTS idx_books_created_at_id ON books (created_at DESC, id DESC);

-- migrate:down
DROP INDEX IF EXISTS idx_books_created_at_id;
CREATE INDEX IF NOT EXISTS idx_books_created_at ON books (created_at DESC);
