-- migrate:up
CREATE TABLE reading_group_lists (
    group_id    TEXT     NOT NULL REFERENCES reading_groups(id) ON DELETE CASCADE,
    list_id     TEXT     NOT NULL REFERENCES reading_lists(id) ON DELETE CASCADE,
    shared_by   TEXT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shared_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (group_id, list_id)
);

-- migrate:down
DROP TABLE reading_group_lists;
