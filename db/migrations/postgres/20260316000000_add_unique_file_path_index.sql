-- migrate:up
CREATE UNIQUE INDEX IF NOT EXISTS idx_book_files_file_path ON book_files(file_path);

-- migrate:down
DROP INDEX IF EXISTS idx_book_files_file_path;
