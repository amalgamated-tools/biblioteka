-- migrate:up
ALTER TABLE kobo_tokens ADD COLUMN token_hash TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_kobo_tokens_token_hash ON kobo_tokens (token_hash);

-- migrate:down
DROP INDEX IF EXISTS idx_kobo_tokens_token_hash;
-- SQLite does not support DROP COLUMN in older versions; no-op for token_hash.
