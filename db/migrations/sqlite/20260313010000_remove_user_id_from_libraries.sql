-- migrate:up
CREATE TABLE libraries_new (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	name TEXT NOT NULL UNIQUE,
	paths TEXT NOT NULL DEFAULT '[]',
	organization_type TEXT NOT NULL DEFAULT 'book_per_folder',
	monitored INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO libraries_new (id, name, paths, organization_type, monitored, created_at, updated_at)
	SELECT id, name, paths, organization_type, monitored, created_at, updated_at FROM libraries;

DROP TABLE libraries;

ALTER TABLE libraries_new RENAME TO libraries;

-- migrate:down
-- Irreversible: user_id ownership data cannot be restored after removal.
SELECT 1/0;
