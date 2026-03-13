-- migrate:up
ALTER TABLE movies ADD COLUMN providers_fetched_at TIMESTAMP;

-- migrate:down
ALTER TABLE movies DROP COLUMN providers_fetched_at;
