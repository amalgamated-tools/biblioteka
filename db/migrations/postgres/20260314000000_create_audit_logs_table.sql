-- migrate:up
CREATE TABLE IF NOT EXISTS audit_logs (
	id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	user_id TEXT,
	action TEXT NOT NULL,
	entity_type TEXT NOT NULL,
	entity_id TEXT NOT NULL,
	metadata TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs (user_id);

-- migrate:down
DROP TABLE IF EXISTS audit_logs;
