-- migrate:up
DROP INDEX IF EXISTS idx_kobo_tokens_token;
ALTER TABLE kobo_tokens DROP COLUMN token;

-- migrate:down
ALTER TABLE kobo_tokens ADD COLUMN token TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_kobo_tokens_token ON kobo_tokens (token);
