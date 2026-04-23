-- migrate:up
-- Add idx_book_annotations_book_created_at (book_id, created_at ASC) so that
-- ListAnnotationsForBook — which filters by book_id and orders by created_at —
-- can read rows in sort order directly from the index and avoid the
-- USE TEMP B-TREE FOR ORDER BY that the existing index causes.
--
-- EXPLAIN QUERY PLAN before:
--   SEARCH ba USING INDEX idx_book_annotations_book_user (book_id=?)
--   USE TEMP B-TREE FOR ORDER BY
--
-- EXPLAIN QUERY PLAN after:
--   SEARCH ba USING INDEX idx_book_annotations_book_created_at (book_id=?)
--   (no temp B-tree)
--
-- Keep idx_book_annotations_book_user (book_id, user_id). Although
-- ListAnnotationsForBook uses a complex OR / subquery predicate, it still
-- includes an equality check on ba.user_id, so the existing composite index
-- may remain useful unless EXPLAIN / ANALYZE shows it is redundant.
-- The new index also serves the books.id ON DELETE CASCADE FK, while the
-- separate idx_book_annotations_user_id index continues to serve the
-- users.id ON DELETE CASCADE FK.
CREATE INDEX IF NOT EXISTS idx_book_annotations_book_created_at
    ON book_annotations (book_id, created_at ASC);

-- migrate:down
DROP INDEX IF EXISTS idx_book_annotations_book_created_at;
