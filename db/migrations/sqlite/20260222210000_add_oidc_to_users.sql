-- migrate:up
ALTER TABLE users ADD COLUMN oidc_subject TEXT;
CREATE UNIQUE INDEX idx_users_oidc_subject ON users(oidc_subject) WHERE oidc_subject IS NOT NULL;

-- migrate:down
DROP INDEX IF EXISTS idx_users_oidc_subject;
ALTER TABLE users DROP COLUMN oidc_subject;
