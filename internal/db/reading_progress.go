package db

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ReadingStats holds aggregate reading progress information for a user.
type ReadingStats struct {
	TotalTracked  int `json:"total_tracked"`
	TotalFinished int `json:"total_finished"`
}

// ListReadingProgress returns all reading progress entries for a user, ordered
// by last update time descending (most-recently synced first).
func (d *DB) ListReadingProgress(ctx context.Context, userID string) ([]ReadingProgress, error) {
	slog.DebugContext(ctx, "listing reading progress",
		slog.String(otelkeys.UserID, userID),
	)
	rows, err := d.QueryContext(ctx,
		`SELECT `+readingProgressColumns+` FROM reading_progress WHERE user_id = $1 ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query reading progress: %w", err)
	}
	items, err := collectRows(rows, scanReadingProgress)
	if err != nil {
		return nil, fmt.Errorf("scan reading progress: %w", err)
	}
	return items, nil
}

// GetReadingStats returns aggregate reading statistics for a user: total
// documents tracked and how many are finished (percentage >= 0.99).
func (d *DB) GetReadingStats(ctx context.Context, userID string) (ReadingStats, error) {
	slog.DebugContext(ctx, "fetching reading stats",
		slog.String(otelkeys.UserID, userID),
	)
	var stats ReadingStats
	err := d.QueryRowContext(ctx,
		`SELECT
			COUNT(*),
			COUNT(CASE WHEN percentage >= 0.99 THEN 1 END)
		FROM reading_progress
		WHERE user_id = $1`,
		userID,
	).Scan(&stats.TotalTracked, &stats.TotalFinished)
	if err != nil {
		return ReadingStats{}, fmt.Errorf("query reading stats: %w", err)
	}
	return stats, nil
}

// GetReadingStreak returns the current consecutive-day reading streak for a user.
// The streak is the number of calendar days (in UTC) ending today or yesterday
// on which at least one reading progress entry exists. Returns 0 when there is no
// activity or the most-recent activity was before yesterday.
func (d *DB) GetReadingStreak(ctx context.Context, userID string) (int, error) {
	slog.DebugContext(ctx, "fetching reading streak",
		slog.String(otelkeys.UserID, userID),
	)
	items, err := d.ListReadingProgress(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Extract timestamps from items and compute the streak.
	timestamps := make([]time.Time, len(items))
	for i, item := range items {
		timestamps[i] = item.UpdatedAt.Time
	}
	return ComputeReadingStreak(timestamps, time.Now().UTC()), nil
}

// collectUniqueDates returns the distinct UTC calendar dates from timestamps,
// each truncated to midnight UTC. The result is unsorted.
func collectUniqueDates(timestamps []time.Time) []time.Time {
	seen := map[time.Time]struct{}{}
	dates := make([]time.Time, 0, len(timestamps))
	for _, ts := range timestamps {
		day := ts.UTC().Truncate(24 * time.Hour)
		if _, ok := seen[day]; !ok {
			seen[day] = struct{}{}
			dates = append(dates, day)
		}
	}
	return dates
}

// ComputeReadingStreak calculates the current consecutive-day reading streak
// from a slice of timestamps relative to the given reference time now.
// The timestamps need not be sorted or unique. A streak is the number of
// calendar days (in UTC) ending today or yesterday on which at least one
// timestamp exists. Returns 0 when there is no activity or the most-recent
// activity was before yesterday. Pass time.Now().UTC() for production use;
// pass a fixed time in tests for deterministic results.
func ComputeReadingStreak(timestamps []time.Time, now time.Time) int {
	if len(timestamps) == 0 {
		return 0
	}

	dates := collectUniqueDates(timestamps)

	// Sort descending (most recent first).
	slices.SortFunc(dates, func(a, b time.Time) int {
		return b.Compare(a)
	})

	today := now.UTC().Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)

	// The streak must end today or yesterday; otherwise it is broken.
	if dates[0].Before(yesterday) {
		return 0
	}

	// Count backwards from the most-recent date, requiring each successive date
	// to be exactly one day earlier than the previous one.
	streak := 1
	for i := 1; i < len(dates); i++ {
		expected := dates[i-1].Add(-24 * time.Hour)
		if !dates[i].Equal(expected) {
			break
		}
		streak++
	}
	return streak
}

// ComputeLongestStreak returns the length of the longest consecutive run of
// calendar days (UTC) in the given slice of timestamps. The timestamps need
// not be sorted or unique. Returns 0 for an empty slice.
func ComputeLongestStreak(timestamps []time.Time) int {
	if len(timestamps) == 0 {
		return 0
	}

	dates := collectUniqueDates(timestamps)

	// Sort ascending (oldest first) to walk forward through consecutive days.
	slices.SortFunc(dates, func(a, b time.Time) int {
		return a.Compare(b)
	})

	longest := 1
	current := 1
	for i := 1; i < len(dates); i++ {
		if dates[i].Equal(dates[i-1].Add(24 * time.Hour)) {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 1
		}
	}
	return longest
}

// YearInBooks holds aggregate reading and download statistics for a calendar year.
type YearInBooks struct {
	Year           int `json:"year"`
	BooksFinished  int `json:"books_finished"`
	ActiveDays     int `json:"active_days"`
	LongestStreak  int `json:"longest_streak"`
	TotalDownloads int `json:"total_downloads"`
}

// GetYearInBooks returns aggregate reading and download statistics for the given
// calendar year and user. BooksFinished counts documents whose percentage
// reached >= 0.99 and were last updated during the requested year.
// ActiveDays is the number of distinct calendar days (UTC) with reading
// activity in that year. LongestStreak is the longest consecutive-day reading
// run within the year. TotalDownloads counts book file downloads in that year.
func (d *DB) GetYearInBooks(ctx context.Context, userID string, year int) (YearInBooks, error) {
	slog.DebugContext(ctx, "db: fetching year-in-books stats",
		slog.String(otelkeys.UserID, userID),
		slog.Int(otelkeys.Year, year),
	)

	result := YearInBooks{Year: year}

	// For Postgres, pass time.Time values in UTC so the driver sends proper
	// timestamptz parameters. For SQLite, use DATETIME-formatted strings so
	// string comparisons work correctly at year boundaries.
	var yearStart, yearEnd any
	if d.Dialect == DialectPostgres {
		yearStart = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		yearEnd = time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		yearStart = fmt.Sprintf("%04d-01-01 00:00:00", year)
		yearEnd = fmt.Sprintf("%04d-01-01 00:00:00", year+1)
	}

	finishedQuery := `
		SELECT COUNT(*) FROM reading_progress
		WHERE user_id = $1
		  AND percentage >= 0.99
		  AND updated_at >= $2 AND updated_at < $3`
	downloadsQuery := `
		SELECT COUNT(*) FROM book_downloads
		WHERE user_id = $1
		  AND downloaded_at >= $2 AND downloaded_at < $3`

	// Postgres TIMESTAMPTZ needs explicit UTC cast for correct day boundaries;
	// SQLite stores text so plain DATE() is fine.
	var activeDaysQuery string
	if d.Dialect == DialectPostgres {
		activeDaysQuery = `
			SELECT COUNT(DISTINCT DATE(updated_at AT TIME ZONE 'UTC'))
			FROM reading_progress
			WHERE user_id = $1
			  AND updated_at >= $2 AND updated_at < $3`
	} else {
		activeDaysQuery = `
			SELECT COUNT(DISTINCT DATE(updated_at))
			FROM reading_progress
			WHERE user_id = $1
			  AND updated_at >= $2 AND updated_at < $3`
	}

	if err := d.QueryRowContext(ctx, finishedQuery, userID, yearStart, yearEnd).Scan(&result.BooksFinished); err != nil {
		return YearInBooks{}, fmt.Errorf("query books finished: %w", err)
	}
	if err := d.QueryRowContext(ctx, activeDaysQuery, userID, yearStart, yearEnd).Scan(&result.ActiveDays); err != nil {
		return YearInBooks{}, fmt.Errorf("query active days: %w", err)
	}
	if err := d.QueryRowContext(ctx, downloadsQuery, userID, yearStart, yearEnd).Scan(&result.TotalDownloads); err != nil {
		return YearInBooks{}, fmt.Errorf("query total downloads: %w", err)
	}

	// Compute longest streak from reading progress timestamps within the year.
	tsQuery := `
		SELECT updated_at FROM reading_progress
		WHERE user_id = $1
		  AND updated_at >= $2 AND updated_at < $3`
	rows, err := d.QueryContext(ctx, tsQuery, userID, yearStart, yearEnd)
	if err != nil {
		return YearInBooks{}, fmt.Errorf("query timestamps for streak: %w", err)
	}
	var timestamps []time.Time
	for rows.Next() {
		var ts Timestamp
		if err := rows.Scan(&ts); err != nil {
			_ = rows.Close()
			return YearInBooks{}, fmt.Errorf("scan timestamp: %w", err)
		}
		timestamps = append(timestamps, ts.Time)
	}
	if err := rows.Close(); err != nil {
		return YearInBooks{}, fmt.Errorf("close rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return YearInBooks{}, fmt.Errorf("rows error: %w", err)
	}

	result.LongestStreak = ComputeLongestStreak(timestamps)
	return result, nil
}
