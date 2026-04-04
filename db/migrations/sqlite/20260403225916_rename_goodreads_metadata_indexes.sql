-- migrate:up
DROP INDEX IF EXISTS idx_goodreads_metadata_user_created_at_id;
DROP INDEX IF EXISTS idx_goodreads_metadata_user_status_created_at_id;

CREATE INDEX IF NOT EXISTS idx_goodreads_metadata_user_status_created_at_id_desc
	ON goodreads_metadata (user_id, status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_goodreads_metadata_user_created_at_id_desc
	ON goodreads_metadata (user_id, created_at DESC, id DESC);

-- migrate:down
DROP INDEX IF EXISTS idx_goodreads_metadata_user_created_at_id_desc;
DROP INDEX IF EXISTS idx_goodreads_metadata_user_status_created_at_id_desc;

CREATE INDEX IF NOT EXISTS idx_goodreads_metadata_user_status_created_at_id
	ON goodreads_metadata (user_id, status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_goodreads_metadata_user_created_at_id
	ON goodreads_metadata (user_id, created_at DESC, id DESC);
