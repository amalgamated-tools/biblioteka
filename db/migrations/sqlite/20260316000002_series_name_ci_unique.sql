-- migrate:up
CREATE UNIQUE INDEX idx_series_name_ci ON series (LOWER(name));

-- migrate:down
DROP INDEX IF EXISTS idx_series_name_ci;
