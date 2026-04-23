-- migrate:up

-- reading_group_members: ListGroupMembers uses
--   WHERE group_id = $1 ORDER BY joined_at ASC
-- The PK autoindex (group_id, user_id) handles the filter but forces a temp B-tree
-- for the joined_at sort. A composite (group_id, joined_at) index eliminates it.
CREATE INDEX IF NOT EXISTS idx_reading_group_members_group_joined_at
    ON reading_group_members (group_id, joined_at);

-- audit_logs: ListAuditLogs uses ORDER BY created_at DESC, id DESC
-- The existing idx_audit_logs_created_at (created_at DESC) covers the first column but
-- forces "USE TEMP B-TREE FOR RIGHT PART OF ORDER BY" for the id DESC tiebreaker.
-- Expanding to (created_at DESC, id DESC) covers both columns.
DROP INDEX IF EXISTS idx_audit_logs_created_at;
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at_id
    ON audit_logs (created_at DESC, id DESC);

-- migrate:down

DROP INDEX IF EXISTS idx_reading_group_members_group_joined_at;

DROP INDEX IF EXISTS idx_audit_logs_created_at_id;
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at
    ON audit_logs (created_at DESC);
