-- migrate:up
CREATE UNIQUE INDEX idx_authors_name_ci ON authors (LOWER(name));

-- migrate:down
DROP INDEX IF EXISTS idx_authors_name_ci;
