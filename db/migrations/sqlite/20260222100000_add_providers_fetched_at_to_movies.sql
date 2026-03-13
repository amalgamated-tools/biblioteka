-- migrate:up
ALTER TABLE movies ADD COLUMN providers_fetched_at DATETIME;

-- migrate:down
ALTER TABLE movies DROP COLUMN providers_fetched_at;
