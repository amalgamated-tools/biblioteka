-- migrate:up
CREATE TABLE tags (
    id TEXT NOT NULL DEFAULT (lower(hex(randomblob(16)))) PRIMARY KEY,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE UNIQUE INDEX idx_tags_name ON tags (LOWER(name));

CREATE TABLE book_tags (
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (book_id, tag_id)
);

-- migrate:down
DROP TABLE IF EXISTS book_tags;
DROP TABLE IF EXISTS tags;
