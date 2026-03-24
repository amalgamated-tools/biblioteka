-- migrate:up
CREATE TABLE IF NOT EXISTS "schema_migrations" (
	version varchar(128) primary key,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- migrate:down
DROP TABLE IF EXISTS schema_migrations;
