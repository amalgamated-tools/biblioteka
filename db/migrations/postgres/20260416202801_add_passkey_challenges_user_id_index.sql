-- migrate:no-transaction
-- migrate:up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_passkey_challenges_user_id ON passkey_challenges (user_id);

-- migrate:down
DROP INDEX IF EXISTS idx_passkey_challenges_user_id;
