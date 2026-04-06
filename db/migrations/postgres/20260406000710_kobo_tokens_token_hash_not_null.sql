-- migrate:up
ALTER TABLE kobo_tokens ALTER COLUMN token_hash SET NOT NULL;

-- migrate:down
ALTER TABLE kobo_tokens ALTER COLUMN token_hash DROP NOT NULL;
