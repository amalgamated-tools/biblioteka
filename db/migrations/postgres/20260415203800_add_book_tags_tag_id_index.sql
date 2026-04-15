-- migrate:up
-- Add an index on book_tags.tag_id to support efficient cascade deletes when a
-- tag is removed, and to accelerate future books-by-tag queries.
-- Without this index, deleting a tag causes a full table scan of book_tags.
CREATE INDEX IF NOT EXISTS idx_book_tags_tag_id ON book_tags (tag_id);

-- migrate:down
DROP INDEX IF EXISTS idx_book_tags_tag_id;
