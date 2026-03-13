-- migrate:up
CREATE TABLE IF NOT EXISTS libraries (
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

-- migrate:down
DROP TABLE IF EXISTS libraries;
