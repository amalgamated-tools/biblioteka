-- migrate:up
-- Add (reading_list_id, added_at) index on reading_list_books to eliminate the
-- temp B-tree in ListReadingListBooks (ORDER BY rlb.added_at ASC, b.id ASC).
-- Without this index PostgreSQL sorts the full result set for the ORDER BY.
-- With this index PostgreSQL can use an index scan in the desired sort order.
CREATE INDEX IF NOT EXISTS idx_reading_list_books_list_added_at
    ON reading_list_books (reading_list_id, added_at ASC);

-- migrate:down
DROP INDEX IF EXISTS idx_reading_list_books_list_added_at;
