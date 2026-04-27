-- migrate:up
-- Add idx_book_annotations_book_created_at (book_id, created_at ASC) so that
-- ListAnnotationsForBook — which filters by book_id and orders by created_at —
-- can use an index scan in sort order and avoid a filesort.
--
-- Keep idx_book_annotations_book_user (book_id, user_id). Although
-- ListAnnotationsForBook uses a complex OR / subquery predicate, it still
-- includes an equality check on ba.user_id, so the existing composite index
-- may remain useful for PostgreSQL plans (for example, bitmap OR / bitmap
-- index combinations) unless EXPLAIN / ANALYZE shows it is redundant.
-- The new index also serves the books.id ON DELETE CASCADE FK, while the
-- separate idx_book_annotations_user_id index continues to serve the
-- users.id ON DELETE CASCADE FK.
CREATE INDEX IF NOT EXISTS idx_book_annotations_book_created_at
    ON book_annotations (book_id, created_at ASC);

-- migrate:down
DROP INDEX IF EXISTS idx_book_annotations_book_created_at;
