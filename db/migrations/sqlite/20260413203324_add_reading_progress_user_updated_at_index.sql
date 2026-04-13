-- migrate:up
CREATE INDEX IF NOT EXISTS idx_reading_progress_user_updated_at ON reading_progress (user_id, updated_at DESC);

-- migrate:down
DROP INDEX IF EXISTS idx_reading_progress_user_updated_at;
