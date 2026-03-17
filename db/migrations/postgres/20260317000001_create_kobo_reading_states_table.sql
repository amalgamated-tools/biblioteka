-- migrate:up
CREATE TABLE IF NOT EXISTS kobo_reading_states (
	id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
	status TEXT NOT NULL DEFAULT 'ReadyToRead',
	percent_read DOUBLE PRECISION,
	location_value TEXT,
	location_type TEXT,
	location_source TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	UNIQUE (user_id, book_id)
);

-- migrate:down
DROP TABLE IF EXISTS kobo_reading_states;
