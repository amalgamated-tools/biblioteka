package db

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// recommendationsQuery is the CTE-based scoring query used by GetRecommendations.
// It works for both SQLite and PostgreSQL.
//
// Scoring weights:
//   - +3 per shared author with a read/reading book
//   - +5 per series continuation (book is the logical next in a series the user is reading)
//   - +1 if the publisher matches any publisher from the user's read books
//   - +download_count/100 as a popularity tiebreaker
//
// When the user has no reading history, all candidate books score 0 and are
// returned ordered by created_at DESC (most-recently-added first), which acts
// as a natural fallback.
const recommendationsQuery = `
WITH
user_reads AS (
    SELECT book_id
    FROM kobo_reading_states
    WHERE user_id = $1
      AND status IN ('Reading', 'Finished')
),
user_authors AS (
    SELECT DISTINCT ba.author_id
    FROM book_authors ba
    INNER JOIN user_reads ur ON ur.book_id = ba.book_id
),
user_publishers AS (
    SELECT DISTINCT b.publisher
    FROM books b
    INNER JOIN user_reads ur ON ur.book_id = b.id
    WHERE b.publisher IS NOT NULL
),
series_progress AS (
    SELECT bs.series_id, MAX(bs.position) AS max_position
    FROM book_series bs
    INNER JOIN user_reads ur ON ur.book_id = bs.book_id
    WHERE bs.position IS NOT NULL
    GROUP BY bs.series_id
),
candidate_scores AS (
    SELECT
        b.id,
        COALESCE((
            SELECT CAST(COUNT(*) AS REAL) * 3.0
            FROM book_authors ba2
            INNER JOIN user_authors ua ON ua.author_id = ba2.author_id
            WHERE ba2.book_id = b.id
        ), 0.0)
        + COALESCE((
            SELECT CAST(COUNT(DISTINCT bs2.series_id) AS REAL) * 5.0
            FROM book_series bs2
            INNER JOIN series_progress sp ON sp.series_id = bs2.series_id
            WHERE bs2.book_id = b.id
              AND bs2.position IS NOT NULL
              AND bs2.position > sp.max_position
        ), 0.0)
        + COALESCE((
            SELECT 1.0
            FROM user_publishers up
            WHERE b.publisher = up.publisher
            LIMIT 1
        ), 0.0)
        + COALESCE((
            SELECT SUM(CAST(bf.download_count AS REAL)) / 100.0
            FROM book_files bf
            WHERE bf.book_id = b.id
        ), 0.0)
        AS score
    FROM books b
    WHERE b.id NOT IN (SELECT book_id FROM user_reads)
)
SELECT b.id, b.title, b.description, b.asin, b.isbn10, b.isbn13,
       b.goodreads_id, b.hardcover_id, b.google_books_id,
       b.publication_date, b.publisher, b.language, b.cover_image_url,
       b.created_at, b.updated_at
FROM candidate_scores cs
INNER JOIN books b ON b.id = cs.id
ORDER BY cs.score DESC, b.created_at DESC
LIMIT $2`

// GetRecommendations returns up to limit books recommended for the given user,
// scored by author overlap, series continuation, publisher match, and download
// popularity. When the user has no reading history every candidate scores 0 and
// results are ordered newest-first, providing a useful default.
func (d *DB) GetRecommendations(ctx context.Context, userID string, limit int) ([]Book, error) {
	slog.DebugContext(ctx, "db: fetching recommendations",
		slog.String(otelkeys.UserID, userID),
		slog.Int(otelkeys.Limit, limit),
	)

	rows, err := d.QueryContext(ctx, recommendationsQuery, userID, limit)
	if err != nil {
		return nil, err
	}
	books, err := collectRows(rows, scanBook)
	if err != nil {
		return nil, err
	}
	return books, nil
}
