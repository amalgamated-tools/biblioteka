-- migrate:up

-- pg_trgm enables GIN trigram indexes that make ILIKE '%query%' use an index
-- instead of a full sequential scan. The extension is bundled with standard
-- PostgreSQL installations and requires no additional dependencies.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- NOTE: Non-concurrent GIN index builds hold a ShareLock on the table for the
-- duration of the build. For libraries with a very large number of books this
-- may cause a brief write pause during the migration. CREATE INDEX CONCURRENTLY
-- would avoid this but cannot run inside a transaction (which dbmate uses),
-- so we accept the trade-off here.
CREATE INDEX idx_books_title_trgm       ON books USING GIN (title gin_trgm_ops);
CREATE INDEX idx_books_description_trgm ON books USING GIN (description gin_trgm_ops);

-- migrate:down
-- Note: We intentionally do not DROP EXTENSION pg_trgm here. The extension is
-- a shared database resource and may be used by other schemas or applications
-- in the same PostgreSQL instance. Removing it could silently break unrelated
-- trigram indexes or queries.
DROP INDEX IF EXISTS idx_books_description_trgm;
DROP INDEX IF EXISTS idx_books_title_trgm;
