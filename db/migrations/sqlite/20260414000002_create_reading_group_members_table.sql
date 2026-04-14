-- migrate:up
CREATE TABLE reading_group_members (
    group_id    TEXT     NOT NULL REFERENCES reading_groups(id) ON DELETE CASCADE,
    user_id     TEXT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT     NOT NULL DEFAULT 'member',
    joined_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX idx_reading_group_members_user ON reading_group_members(user_id);

-- migrate:down
DROP INDEX IF EXISTS idx_reading_group_members_user;
DROP TABLE reading_group_members;
