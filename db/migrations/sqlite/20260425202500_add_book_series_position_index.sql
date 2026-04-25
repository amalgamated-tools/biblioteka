-- migrate:up
-- Add composite (series_id, position ASC) index on book_series.
-- ListBooksBySeries and ListBooksBySeriesPaginated filter on series_id and sort
-- by position ASC. The existing idx_book_series_series_id covers the filter but
-- leaves the position sort to a full temp B-tree. This composite index lets
-- SQLite walk rows already in position order, reducing the sort to only the
-- title tiebreaker (RIGHT PART OF ORDER BY).
CREATE INDEX IF NOT EXISTS idx_book_series_series_position ON book_series (series_id, position ASC);

-- migrate:down
DROP INDEX IF EXISTS idx_book_series_series_position;
