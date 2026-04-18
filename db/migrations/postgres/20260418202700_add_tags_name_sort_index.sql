-- migrate:up
CREATE INDEX IF NOT EXISTS idx_tags_name_sort ON tags(name);

-- migrate:down
DROP INDEX IF EXISTS idx_tags_name_sort;
