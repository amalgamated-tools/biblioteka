-- migrate:up
-- Add (reading_list_id, added_at, book_id) index on reading_list_books to
-- support ListReadingListBooks (ORDER BY rlb.added_at ASC, b.id ASC).
-- Without this index PostgreSQL sorts the matching rows for the ORDER BY.
-- With this index PostgreSQL can scan rows in the desired sort order using
-- book_id (= b.id) as the tiebreaker column.
CREATE INDEX IF NOT EXISTS idx_reading_list_books_list_added_at
    ON reading_list_books (reading_list_id, added_at ASC, book_id ASC);

-- migrate:down
DROP INDEX IF EXISTS idx_reading_list_books_list_added_at;
