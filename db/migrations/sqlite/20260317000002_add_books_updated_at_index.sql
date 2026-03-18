-- migrate:up
CREATE INDEX IF NOT EXISTS idx_books_updated_at_id ON books (updated_at, id);

-- migrate:down
DROP INDEX IF EXISTS idx_books_updated_at_id;
