-- migrate:up

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reading_group_members_group_joined_at
    ON reading_group_members (group_id, joined_at);

DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_created_at;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_created_at_id
    ON audit_logs (created_at DESC, id DESC);

-- migrate:down

DROP INDEX CONCURRENTLY IF EXISTS idx_reading_group_members_group_joined_at;

DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_created_at_id;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_created_at
    ON audit_logs (created_at DESC);
