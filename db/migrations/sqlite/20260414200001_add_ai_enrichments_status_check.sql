-- migrate:up
-- SQLite does not support ALTER TABLE ADD CONSTRAINT, so we recreate the table
-- with a CHECK constraint on the status column.
CREATE TABLE ai_enrichments_new (
    id TEXT NOT NULL DEFAULT (lower(hex(randomblob(16)))) PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id TEXT REFERENCES books(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'applied', 'rejected')),
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    suggested_tags TEXT NOT NULL DEFAULT '[]',
    reading_level TEXT,
    generated_description TEXT,
    raw_response TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO ai_enrichments_new SELECT * FROM ai_enrichments;
DROP TABLE ai_enrichments;
ALTER TABLE ai_enrichments_new RENAME TO ai_enrichments;

CREATE INDEX idx_ai_enrichments_user_book_status ON ai_enrichments (user_id, book_id, status, created_at DESC);

-- migrate:down
CREATE TABLE ai_enrichments_new (
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

INSERT INTO ai_enrichments_new SELECT * FROM ai_enrichments;
DROP TABLE ai_enrichments;
ALTER TABLE ai_enrichments_new RENAME TO ai_enrichments;

CREATE INDEX idx_ai_enrichments_user_book_status ON ai_enrichments (user_id, book_id, status, created_at DESC);
