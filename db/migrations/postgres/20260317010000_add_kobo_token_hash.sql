-- migrate:up
ALTER TABLE kobo_tokens ADD COLUMN IF NOT EXISTS token_hash TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_kobo_tokens_token_hash ON kobo_tokens (token_hash);

-- migrate:down
DROP INDEX IF EXISTS idx_kobo_tokens_token_hash;
ALTER TABLE kobo_tokens DROP COLUMN IF EXISTS token_hash;
