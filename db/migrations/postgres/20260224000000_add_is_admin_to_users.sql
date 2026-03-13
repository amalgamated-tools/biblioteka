-- migrate:up
ALTER TABLE users ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT FALSE;

-- migrate:down
ALTER TABLE users DROP COLUMN is_admin;
