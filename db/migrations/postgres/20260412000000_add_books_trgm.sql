-- migrate:up

-- pg_trgm enables GIN trigram indexes that make ILIKE '%query%' use an index
-- instead of a full sequential scan. The extension is bundled with standard
-- PostgreSQL installations and requires no additional dependencies.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_books_title_trgm       ON books USING GIN (title gin_trgm_ops);
CREATE INDEX idx_books_description_trgm ON books USING GIN (description gin_trgm_ops);

-- migrate:down
DROP INDEX IF EXISTS idx_books_description_trgm;
DROP INDEX IF EXISTS idx_books_title_trgm;
DROP EXTENSION IF EXISTS pg_trgm;
