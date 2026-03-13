-- migrate:up
CREATE TABLE IF NOT EXISTS libraries (
	id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	paths TEXT NOT NULL DEFAULT '[]',
	organization_type TEXT NOT NULL DEFAULT 'book_per_folder',
	monitored BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	UNIQUE(user_id, name)
);

CREATE INDEX idx_libraries_user_id ON libraries(user_id);

-- migrate:down
DROP TABLE IF EXISTS libraries;
