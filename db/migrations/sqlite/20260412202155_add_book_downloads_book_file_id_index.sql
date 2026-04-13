-- migrate:up
CREATE INDEX IF NOT EXISTS idx_book_downloads_book_file_id ON book_downloads (book_file_id);

-- migrate:down
DROP INDEX IF EXISTS idx_book_downloads_book_file_id;
