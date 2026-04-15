-- migrate:up
CREATE TABLE tags (
    id TEXT NOT NULL DEFAULT gen_random_uuid()::text PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
