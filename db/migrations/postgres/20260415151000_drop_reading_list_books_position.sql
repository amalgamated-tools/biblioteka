-- migrate:up
ALTER TABLE reading_list_books DROP COLUMN position;

-- migrate:down
ALTER TABLE reading_list_books ADD COLUMN position INTEGER NOT NULL DEFAULT 0;
