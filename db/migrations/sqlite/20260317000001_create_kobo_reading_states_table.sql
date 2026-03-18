-- migrate:up
CREATE TABLE IF NOT EXISTS kobo_reading_states (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
	status TEXT NOT NULL DEFAULT 'ReadyToRead',
	percent_read REAL,
	location_value TEXT,
	location_type TEXT,
	location_source TEXT,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
	UNIQUE (user_id, book_id)
);

CREATE INDEX IF NOT EXISTS idx_kobo_reading_states_user_updated ON kobo_reading_states (user_id, updated_at);

-- migrate:down
DROP TABLE IF EXISTS kobo_reading_states;
