-- migrate:up
ALTER TABLE goodreads_metadata ADD COLUMN hardcover_id TEXT;
ALTER TABLE goodreads_metadata ADD COLUMN google_books_id TEXT;

-- migrate:down
ALTER TABLE goodreads_metadata DROP COLUMN google_books_id;
ALTER TABLE goodreads_metadata DROP COLUMN hardcover_id;
