-- migrate:up
CREATE TABLE reading_lists (
    id          TEXT     NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id     TEXT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT     NOT NULL,
    description TEXT,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX idx_reading_lists_user_name ON reading_lists(user_id, name);

-- migrate:down
DROP TABLE reading_lists;
