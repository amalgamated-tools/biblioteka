-- migrate:up
CREATE INDEX idx_ai_enrichments_user_book_status ON ai_enrichments (user_id, book_id, status, created_at DESC);

-- migrate:down
DROP INDEX IF EXISTS idx_ai_enrichments_user_book_status;
