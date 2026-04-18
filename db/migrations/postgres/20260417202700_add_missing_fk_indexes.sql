-- migrate:up
-- kobo_reading_states.book_id: ON DELETE CASCADE from books — no index causes full table scan
CREATE INDEX IF NOT EXISTS idx_kobo_reading_states_book_id ON kobo_reading_states (book_id);

-- ai_enrichments.book_id: ON DELETE SET NULL from books — no index causes full table scan
CREATE INDEX IF NOT EXISTS idx_ai_enrichments_book_id ON ai_enrichments (book_id);

-- goodreads_metadata.book_id: ON DELETE SET NULL from books — no index causes full table scan
CREATE INDEX IF NOT EXISTS idx_goodreads_metadata_book_id ON goodreads_metadata (book_id);

-- book_annotations.user_id: ON DELETE CASCADE from users — no index causes full table scan
CREATE INDEX IF NOT EXISTS idx_book_annotations_user_id ON book_annotations (user_id);

-- migrate:down
DROP INDEX IF EXISTS idx_book_annotations_user_id;
DROP INDEX IF EXISTS idx_goodreads_metadata_book_id;
DROP INDEX IF EXISTS idx_ai_enrichments_book_id;
DROP INDEX IF EXISTS idx_kobo_reading_states_book_id;
