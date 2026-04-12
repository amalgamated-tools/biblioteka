package db

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// MonthlyDownloadCount holds the number of book-file downloads for a single
// calendar month, expressed as a YYYY-MM string.
type MonthlyDownloadCount struct {
	Month string `json:"month"` // "YYYY-MM"
	Count int    `json:"count"`
}

// RecordBookDownload inserts a timestamped download event for the given book
// file and user. The error is non-critical and callers should log and continue
// rather than surfacing it to the end user.
func (d *DB) RecordBookDownload(ctx context.Context, bookFileID, userID string) error {
	slog.DebugContext(ctx, "db: recording book download",
		slog.String(otelkeys.BookFileID, bookFileID),
		slog.String(otelkeys.UserID, userID),
	)
	_, err := d.ExecContext(ctx,
		`INSERT INTO book_downloads (book_file_id, user_id) VALUES ($1, $2)`,
		bookFileID, userID,
	)
	return err
}

// GetMonthlyDownloads returns the download counts per calendar month for the
// authenticated user over the last `months` calendar months (including the
// current month). Results are ordered oldest-first and always contain an entry
// for every month in the window, even if the count is zero.
func (d *DB) GetMonthlyDownloads(ctx context.Context, userID string, months int) ([]MonthlyDownloadCount, error) {
	slog.DebugContext(ctx, "db: fetching monthly download counts",
		slog.String(otelkeys.UserID, userID),
		slog.Int(otelkeys.Count, months),
	)

	var query string
	if d.Dialect == DialectPostgres {
		query = `
SELECT
    TO_CHAR(gs.month, 'YYYY-MM') AS month,
    COALESCE(COUNT(bd.id), 0)    AS count
FROM generate_series(
    DATE_TRUNC('month', NOW() - (($2 - 1) || ' months')::interval),
    DATE_TRUNC('month', NOW()),
    '1 month'::interval
) AS gs(month)
LEFT JOIN book_downloads bd
    ON  DATE_TRUNC('month', bd.downloaded_at) = gs.month
    AND bd.user_id = $1
GROUP BY gs.month
ORDER BY gs.month ASC`
	} else {
		// SQLite: generate a series of months by joining on a cte of integers.
		query = `
WITH RECURSIVE months(n) AS (
    SELECT 0
    UNION ALL
    SELECT n + 1 FROM months WHERE n < $2 - 1
),
month_series AS (
    SELECT strftime('%Y-%m',
        date(
            date('now', 'start of month'),
            '-' || (($2 - 1) - n) || ' months'
        )
    ) AS month
    FROM months
)
SELECT
    ms.month,
    COUNT(bd.id) AS count
FROM month_series ms
LEFT JOIN book_downloads bd
    ON  strftime('%Y-%m', bd.downloaded_at) = ms.month
    AND bd.user_id = $1
GROUP BY ms.month
ORDER BY ms.month ASC`
	}

	rows, err := d.QueryContext(ctx, query, userID, months)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]MonthlyDownloadCount, 0)
	for rows.Next() {
		var m MonthlyDownloadCount
		if err := rows.Scan(&m.Month, &m.Count); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
