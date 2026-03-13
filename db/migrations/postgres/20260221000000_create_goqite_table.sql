-- migrate:up
-- goqite table not used with PostgreSQL; job queue uses Redis via asynq
SELECT 1;
-- migrate:down
-- no-op
