-- SQLite-only migration.
--
-- The PostgreSQL equivalent of this migration is not needed: the initial
-- goodreads_metadata table migration for PostgreSQL
-- (20260323120001_create_goodreads_metadata_table.sql) already created these
-- indexes with the _desc suffix from the start.
--
-- Both dialects end up with the same index names after this migration runs on
-- SQLite. However, because PostgreSQL never applied this migration, the
-- schema_migrations table will show a divergence (PostgreSQL missing
-- 20260403225916). This is expected and intentional — the schema end-state is
-- identical across both dialects.

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
