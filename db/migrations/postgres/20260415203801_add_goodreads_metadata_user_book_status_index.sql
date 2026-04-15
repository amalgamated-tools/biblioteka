-- migrate:up
-- Add a composite index covering the three equality predicates in
-- GetLatestGoodreadsMetadataForBook:
--   WHERE user_id = $1 AND book_id = $2 AND status = $3
--   ORDER BY created_at DESC, id DESC LIMIT 1
--
-- The existing idx_goodreads_metadata_user_status_created_at_id_desc only
-- covers (user_id, status), requiring a scan of all rows for that user+status
-- combination to find the matching book_id.  The new index covers all three
-- equality columns plus the ordering columns, turning it into a point lookup.
CREATE INDEX IF NOT EXISTS idx_goodreads_metadata_user_book_status
    ON goodreads_metadata (user_id, book_id, status, created_at DESC, id DESC);

-- migrate:down
DROP INDEX IF EXISTS idx_goodreads_metadata_user_book_status;
