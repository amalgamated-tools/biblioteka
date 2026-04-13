-- migrate:up
CREATE TABLE reading_lists (
    id          TEXT      NOT NULL PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id     TEXT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT      NOT NULL,
    description TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_reading_lists_user_name ON reading_lists(user_id, name);

-- migrate:down
DROP TABLE reading_lists;
