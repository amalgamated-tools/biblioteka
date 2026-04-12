-- migrate:up
CREATE TABLE IF NOT EXISTS book_downloads (
	id            TEXT        PRIMARY KEY DEFAULT encode(gen_random_bytes(16), 'hex'),
	book_file_id  TEXT        NOT NULL REFERENCES book_files(id) ON DELETE CASCADE,
	user_id       TEXT        NOT NULL REFERENCES users(id)      ON DELETE CASCADE,
	downloaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_book_downloads_user_id        ON book_downloads(user_id);
CREATE INDEX idx_book_downloads_downloaded_at  ON book_downloads(downloaded_at);

-- migrate:down
DROP TABLE IF EXISTS book_downloads;
