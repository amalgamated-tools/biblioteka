-- migrate:up
CREATE TABLE ai_enrichments (
    id TEXT NOT NULL DEFAULT (lower(hex(randomblob(16)))) PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id TEXT REFERENCES books(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    suggested_tags TEXT NOT NULL DEFAULT '[]',
    reading_level TEXT,
    generated_description TEXT,
    raw_response TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- migrate:down
DROP TABLE IF EXISTS ai_enrichments;
