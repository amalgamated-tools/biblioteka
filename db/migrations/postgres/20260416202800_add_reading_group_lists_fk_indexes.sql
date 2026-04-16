-- migrate:up
CREATE INDEX IF NOT EXISTS idx_reading_group_lists_list_id ON reading_group_lists (list_id);
CREATE INDEX IF NOT EXISTS idx_reading_group_lists_shared_by ON reading_group_lists (shared_by);

-- migrate:down
DROP INDEX IF EXISTS idx_reading_group_lists_shared_by;
DROP INDEX IF EXISTS idx_reading_group_lists_list_id;
