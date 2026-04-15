package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// recommendationsQuery is the CTE-based scoring query used by GetRecommendations.
// It works for both SQLite and PostgreSQL.
//
// Scoring weights:
//   - +3 per shared author with a read/reading book
//   - +5 per series continuation (book is the immediate next in a series the user is reading)
//   - +1 if the publisher matches any publisher from the user's read books
//   - +download_count/100 as a popularity tiebreaker
//
// When the user has no reading history, all candidate books score 0 and are
// returned ordered by created_at DESC (most-recently-added first), which acts
// as a natural fallback.
var recommendationsQuery = fmt.Sprintf(`
WITH
user_reads AS (
    SELECT book_id
    FROM kobo_reading_states
    WHERE user_id = $1
      AND status IN ('%s', '%s')
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
author_pts AS (
    SELECT ba.book_id, CAST(COUNT(*) AS REAL) * 3.0 AS pts
    FROM book_authors ba
    INNER JOIN user_authors ua ON ua.author_id = ba.author_id
    GROUP BY ba.book_id
),
series_pts AS (
    SELECT bs.book_id, CAST(COUNT(DISTINCT bs.series_id) AS REAL) * 5.0 AS pts
    FROM book_series bs
    INNER JOIN series_progress sp ON sp.series_id = bs.series_id
    WHERE bs.position IS NOT NULL
      AND bs.position > sp.max_position
      AND NOT EXISTS (
          SELECT 1
          FROM book_series bs2
          WHERE bs2.series_id = bs.series_id
            AND bs2.position > sp.max_position
            AND bs2.position < bs.position
      )
    GROUP BY bs.book_id
),
publisher_pts AS (
    SELECT DISTINCT b.id AS book_id, 1.0 AS pts
    FROM books b
    INNER JOIN user_publishers up ON b.publisher = up.publisher
),
download_pts AS (
    SELECT bf.book_id, SUM(CAST(bf.download_count AS REAL)) / 100.0 AS pts
    FROM book_files bf
    GROUP BY bf.book_id
),
candidate_scores AS (
    SELECT
        b.id,
        COALESCE(ap.pts, 0.0)
        + COALESCE(sp.pts, 0.0)
        + COALESCE(pp.pts, 0.0)
        + COALESCE(dp.pts, 0.0) AS score
    FROM books b
    LEFT JOIN author_pts ap ON ap.book_id = b.id
    LEFT JOIN series_pts sp ON sp.book_id = b.id
    LEFT JOIN publisher_pts pp ON pp.book_id = b.id
    LEFT JOIN download_pts dp ON dp.book_id = b.id
    WHERE NOT EXISTS (SELECT 1 FROM user_reads ur WHERE ur.book_id = b.id)
)
SELECT b.id, b.title, b.description, b.asin, b.isbn10, b.isbn13,
       b.goodreads_id, b.hardcover_id, b.google_books_id,
       b.publication_date, b.publisher, b.language, b.cover_image_url,
       b.created_at, b.updated_at
FROM candidate_scores cs
INNER JOIN books b ON b.id = cs.id
ORDER BY cs.score DESC, b.created_at DESC
LIMIT $2`, StatusReading, StatusFinished)

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
