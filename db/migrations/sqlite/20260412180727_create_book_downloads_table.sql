-- migrate:up
CREATE TABLE IF NOT EXISTS book_downloads (
	id          TEXT     PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	book_file_id TEXT    NOT NULL REFERENCES book_files(id) ON DELETE CASCADE,
	user_id      TEXT    NOT NULL REFERENCES users(id)      ON DELETE CASCADE,
	downloaded_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_book_downloads_user_downloaded ON book_downloads(user_id, downloaded_at);

-- migrate:down
DROP TABLE IF EXISTS book_downloads;
