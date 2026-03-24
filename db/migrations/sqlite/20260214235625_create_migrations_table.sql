-- migrate:up
CREATE TABLE IF NOT EXISTS "schema_migrations" (
	version varchar(128) primary key,
	applied_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- migrate:down
DROP TABLE IF EXISTS schema_migrations;
