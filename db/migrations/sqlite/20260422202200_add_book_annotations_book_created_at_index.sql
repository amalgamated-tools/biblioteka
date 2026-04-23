-- migrate:up
-- Replace idx_book_annotations_book_user (book_id, user_id) with
-- idx_book_annotations_book_created_at (book_id, created_at ASC) so that
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
-- ListAnnotationsForBook does include a ba.user_id = ? predicate, but only as
-- part of a more complex OR / subquery condition. This migration keeps the
-- index change because the SQLite query plan above shows that (book_id,
-- created_at) avoids the temp B-tree for ORDER BY while still serving the
-- book_id lookup and the books.id ON DELETE CASCADE FK.
-- The separate idx_book_annotations_user_id index continues to serve the
-- users.id ON DELETE CASCADE FK.
DROP INDEX IF EXISTS idx_book_annotations_book_user;

CREATE INDEX IF NOT EXISTS idx_book_annotations_book_created_at
    ON book_annotations (book_id, created_at ASC);

-- migrate:down
DROP INDEX IF EXISTS idx_book_annotations_book_created_at;

CREATE INDEX IF NOT EXISTS idx_book_annotations_book_user
    ON book_annotations (book_id, user_id);
