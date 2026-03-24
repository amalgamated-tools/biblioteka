-- migrate:up
ALTER TABLE books DROP COLUMN num_pages;

-- migrate:down
ALTER TABLE books ADD COLUMN num_pages INTEGER;
