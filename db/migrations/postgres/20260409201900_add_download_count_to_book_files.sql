-- migrate:up
ALTER TABLE book_files ADD COLUMN download_count BIGINT NOT NULL DEFAULT 0;

-- migrate:down
ALTER TABLE book_files DROP COLUMN download_count;
