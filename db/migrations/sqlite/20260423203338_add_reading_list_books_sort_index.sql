-- migrate:up
-- Add (reading_list_id, added_at, book_id) index on reading_list_books to
-- eliminate the temp B-tree in ListReadingListBooks
-- (ORDER BY rlb.added_at ASC, b.id ASC).
-- Without this index SQLite scans the PK (reading_list_id, book_id) and then
-- materializes the full result set into a temp B-tree to sort by added_at.
-- With this index SQLite reads rows in the full ORDER BY order directly,
-- avoiding the sort entirely (book_id = b.id covers the tiebreaker column).
CREATE INDEX IF NOT EXISTS idx_reading_list_books_list_added_at
    ON reading_list_books (reading_list_id, added_at ASC, book_id ASC);

-- migrate:down
DROP INDEX IF EXISTS idx_reading_list_books_list_added_at;
