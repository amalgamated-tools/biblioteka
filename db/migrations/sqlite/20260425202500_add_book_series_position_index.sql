-- migrate:up
-- Add composite (series_id, position ASC) index on book_series.
-- ListBooksBySeries and ListBooksBySeriesPaginated filter on series_id and sort
-- by position ASC. This composite index subsumes the old single-column
-- idx_book_series_series_id: SQLite can satisfy any equality filter on series_id
-- alone via the leading column, so the old index is dropped to avoid redundant
-- write overhead and wasted disk space.
CREATE INDEX IF NOT EXISTS idx_book_series_series_position ON book_series (series_id, position ASC);
DROP INDEX IF EXISTS idx_book_series_series_id;

-- migrate:down
CREATE INDEX IF NOT EXISTS idx_book_series_series_id ON book_series (series_id);
DROP INDEX IF EXISTS idx_book_series_series_position;
