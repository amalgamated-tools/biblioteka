-- migrate:up
-- Add composite (series_id, position ASC NULLS LAST) index on book_series.
-- ListBooksBySeries and ListBooksBySeriesPaginated filter on series_id and sort
-- by position ASC NULLS LAST. The existing idx_book_series_series_id covers the
-- filter but leaves the position sort to a full filesort. This composite index
-- lets PostgreSQL walk rows already in position order, reducing the sort to only
-- the title tiebreaker (partial sort on tied positions).
CREATE INDEX IF NOT EXISTS idx_book_series_series_position ON book_series (series_id, position ASC NULLS LAST);

-- migrate:down
DROP INDEX IF EXISTS idx_book_series_series_position;
