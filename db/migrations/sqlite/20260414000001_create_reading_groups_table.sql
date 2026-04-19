-- migrate:up
CREATE TABLE reading_groups (
    id          TEXT     NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    owner_id    TEXT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT     NOT NULL,
    description TEXT,
    created_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE UNIQUE INDEX idx_reading_groups_owner_name ON reading_groups(owner_id, name);

-- migrate:down
DROP INDEX IF EXISTS idx_reading_groups_owner_name;
DROP TABLE reading_groups;
