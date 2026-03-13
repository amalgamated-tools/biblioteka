-- migrate:up
DROP TRIGGER IF EXISTS goqite_updated_timestamp;
DROP INDEX IF EXISTS goqite_queue_priority_created_idx;
DROP TABLE IF EXISTS goqite;

-- migrate:down
CREATE TABLE IF NOT EXISTS goqite (
	id TEXT PRIMARY KEY DEFAULT ('m_' || lower(hex(randomblob(16)))),
	created TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	updated TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	queue TEXT NOT NULL,
	body BLOB NOT NULL,
	timeout TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	received INTEGER NOT NULL DEFAULT 0,
	priority INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE TRIGGER IF NOT EXISTS goqite_updated_timestamp AFTER UPDATE ON goqite BEGIN
	UPDATE goqite SET updated = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = old.id;
END;

CREATE INDEX IF NOT EXISTS goqite_queue_priority_created_idx ON goqite (queue, priority DESC, created);
