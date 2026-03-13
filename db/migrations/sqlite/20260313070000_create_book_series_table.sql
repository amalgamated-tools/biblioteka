-- migrate:up
CREATE TABLE IF NOT EXISTS book_series (
	book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
	series_id TEXT NOT NULL REFERENCES series(id) ON DELETE CASCADE,
	position REAL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	PRIMARY KEY (book_id, series_id)
);

CREATE INDEX idx_book_series_series_id ON book_series(series_id);

-- migrate:down
DROP TABLE IF EXISTS book_series;
