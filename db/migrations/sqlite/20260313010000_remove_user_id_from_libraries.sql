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
-- Cannot restore user_id data; down migration is destructive.
DROP TABLE IF EXISTS libraries;
CREATE TABLE libraries (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	paths TEXT NOT NULL DEFAULT '[]',
	organization_type TEXT NOT NULL DEFAULT 'book_per_folder',
	monitored INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
	UNIQUE(user_id, name)
);
CREATE INDEX idx_libraries_user_id ON libraries(user_id);
