-- migrate:up

-- api_keys: replace single-column user_id index with composite (user_id, created_at, id)
-- ListAPIKeys queries WHERE user_id = $1 ORDER BY created_at DESC, id DESC.
-- The composite index covers both the equality filter and the sort, eliminating
-- the temp B-tree that the single-column idx_api_keys_user_id required.
DROP INDEX IF EXISTS idx_api_keys_user_id;
CREATE INDEX IF NOT EXISTS idx_api_keys_user_created_at ON api_keys (user_id, created_at, id);

-- kobo_tokens: same pattern as api_keys
DROP INDEX IF EXISTS idx_kobo_tokens_user_id;
CREATE INDEX IF NOT EXISTS idx_kobo_tokens_user_created_at ON kobo_tokens (user_id, created_at, id);

-- passkey_credentials: same pattern — ListPasskeyCredentials uses user_id + ORDER BY created_at DESC, id DESC
DROP INDEX IF EXISTS idx_passkey_credentials_user_id;
CREATE INDEX IF NOT EXISTS idx_passkey_credentials_user_created_at ON passkey_credentials (user_id, created_at, id);

-- migrate:down

DROP INDEX IF EXISTS idx_api_keys_user_created_at;
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys (user_id);

DROP INDEX IF EXISTS idx_kobo_tokens_user_created_at;
CREATE INDEX IF NOT EXISTS idx_kobo_tokens_user_id ON kobo_tokens (user_id);

DROP INDEX IF EXISTS idx_passkey_credentials_user_created_at;
CREATE INDEX IF NOT EXISTS idx_passkey_credentials_user_id ON passkey_credentials (user_id);
