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
	slog.DebugContext(ctx, "db: listing reading progress",
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
	slog.DebugContext(ctx, "db: fetching reading stats",
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
	slog.DebugContext(ctx, "db: fetching reading streak",
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
	return ComputeReadingStreak(timestamps), nil
}

// ComputeReadingStreak calculates the current consecutive-day reading streak
// from a slice of timestamps. The timestamps need not be sorted or unique.
// A streak is the number of calendar days (in UTC) ending today or yesterday
// on which at least one timestamp exists. Returns 0 when there is no activity
// or the most-recent activity was before yesterday.
func ComputeReadingStreak(timestamps []time.Time) int {
	if len(timestamps) == 0 {
		return 0
	}

	// Collect distinct calendar dates (UTC).
	seen := map[string]struct{}{}
	var dates []time.Time
	for _, ts := range timestamps {
		dayKey := ts.UTC().Format("2006-01-02")
		if _, ok := seen[dayKey]; !ok {
			seen[dayKey] = struct{}{}
			dates = append(dates, ts.UTC().Truncate(24*time.Hour))
		}
	}

	// Sort descending (most recent first).
	slices.SortFunc(dates, func(a, b time.Time) int {
		return b.Compare(a)
	})

	today := time.Now().UTC().Truncate(24 * time.Hour)
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
