-- migrate:up
-- Replace idx_book_annotations_book_user (book_id, user_id) with
-- idx_book_annotations_book_created_at (book_id, created_at ASC) so that
-- ListAnnotationsForBook — which filters by book_id and orders by created_at —
-- can use an index scan in sort order and avoid a filesort.
--
-- The (book_id, user_id) index is no longer needed: the only book_id-filtered
-- query in the codebase is ListAnnotationsForBook, which does not equality-
-- filter on user_id (it uses a complex OR / subquery predicate).
-- The FK cascade for books.id ON DELETE CASCADE is now served by this index.
-- The separate idx_book_annotations_user_id index continues to serve the
-- users.id ON DELETE CASCADE FK.
DROP INDEX IF EXISTS idx_book_annotations_book_user;

CREATE INDEX IF NOT EXISTS idx_book_annotations_book_created_at
    ON book_annotations (book_id, created_at ASC);

-- migrate:down
DROP INDEX IF EXISTS idx_book_annotations_book_created_at;

CREATE INDEX IF NOT EXISTS idx_book_annotations_book_user
    ON book_annotations (book_id, user_id);
