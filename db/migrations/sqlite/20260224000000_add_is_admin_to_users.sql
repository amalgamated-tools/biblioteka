-- migrate:up
ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;

-- migrate:down
ALTER TABLE users DROP COLUMN is_admin;
