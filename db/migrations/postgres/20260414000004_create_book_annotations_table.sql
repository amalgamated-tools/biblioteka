-- migrate:up
CREATE TABLE book_annotations (
    id          TEXT        NOT NULL PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id     TEXT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id     TEXT        NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    text        TEXT        NOT NULL,
    cfi         TEXT,
    group_id    TEXT        REFERENCES reading_groups(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_book_annotations_book_user ON book_annotations(book_id, user_id);
CREATE INDEX idx_book_annotations_group ON book_annotations(group_id);

-- migrate:down
DROP INDEX IF EXISTS idx_book_annotations_group;
DROP INDEX IF EXISTS idx_book_annotations_book_user;
DROP TABLE book_annotations;
