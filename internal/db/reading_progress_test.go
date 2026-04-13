package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// createTestUserForReadingProgress creates a user for reading progress tests.
func createTestUserForReadingProgress(t *testing.T, d *DB, email string) *User {
	t.Helper()
	user, err := d.CreateUser(t.Context(), "Test User", email, "hashedpw")
	require.NoError(t, err, "CreateUser(%q)", email)
	return user
}

// ---- ListReadingProgress tests ----

func TestListReadingProgress_Empty(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForReadingProgress(t, d, "empty@example.com")
	ctx := t.Context()

	items, err := d.ListReadingProgress(ctx, user.ID)
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestListReadingProgress_ReturnsSortedDescending(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForReadingProgress(t, d, "sorted@example.com")
	ctx := t.Context()

	now := time.Now().UTC()

	// Insert rows with explicit timestamps so ordering is deterministic
	// regardless of SQLite's second-level datetime('now') resolution.
	for i, doc := range []string{"doc-a", "doc-b", "doc-c"} {
		ts := now.Add(time.Duration(i) * time.Minute).Format("2006-01-02 15:04:05")
		_, err := d.ExecContext(ctx,
			`INSERT INTO reading_progress (user_id, document, progress, percentage, updated_at)
			 VALUES ($1, $2, '/p[1]', 0.1, $3)`,
			user.ID, doc, ts,
		)
		require.NoError(t, err)
	}

	items, err := d.ListReadingProgress(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, items, 3)

	// Most-recently updated first (doc-c has the latest timestamp).
	require.Equal(t, "doc-c", items[0].Document)
	require.Equal(t, "doc-b", items[1].Document)
	require.Equal(t, "doc-a", items[2].Document)
}

func TestListReadingProgress_IsolatedByUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUserForReadingProgress(t, d, "u1@example.com")
	user2 := createTestUserForReadingProgress(t, d, "u2@example.com")
	ctx := t.Context()

	_, err := d.UpsertReadingProgress(ctx, user1.ID, "shared-doc", "/p[1]", 0.3, nil, nil)
	require.NoError(t, err)

	items, err := d.ListReadingProgress(ctx, user2.ID)
	require.NoError(t, err)
	require.Empty(t, items)
}

// ---- GetReadingStats tests ----

func TestGetReadingStats_NoData(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForReadingProgress(t, d, "stats-empty@example.com")
	ctx := t.Context()

	stats, err := d.GetReadingStats(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 0, stats.TotalTracked)
	require.Equal(t, 0, stats.TotalFinished)
}

func TestGetReadingStats_Categories(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForReadingProgress(t, d, "stats@example.com")
	ctx := t.Context()

	// percentage = 0 → not started (tracked but not in-progress or finished)
	_, err := d.UpsertReadingProgress(ctx, user.ID, "doc-zero", "/p[1]", 0.0, nil, nil)
	require.NoError(t, err)

	// in-progress
	_, err = d.UpsertReadingProgress(ctx, user.ID, "doc-mid", "/p[5]", 0.45, nil, nil)
	require.NoError(t, err)

	// finished (>= 0.99)
	_, err = d.UpsertReadingProgress(ctx, user.ID, "doc-done", "/p[100]", 1.0, nil, nil)
	require.NoError(t, err)

	// also finished (exactly at threshold)
	_, err = d.UpsertReadingProgress(ctx, user.ID, "doc-near-done", "/p[99]", 0.99, nil, nil)
	require.NoError(t, err)

	stats, err := d.GetReadingStats(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 4, stats.TotalTracked)
	require.Equal(t, 2, stats.TotalFinished)
}

func TestGetReadingStats_IsolatedByUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUserForReadingProgress(t, d, "stats-u1@example.com")
	user2 := createTestUserForReadingProgress(t, d, "stats-u2@example.com")
	ctx := t.Context()

	_, err := d.UpsertReadingProgress(ctx, user1.ID, "doc-a", "/p[1]", 0.5, nil, nil)
	require.NoError(t, err)

	stats, err := d.GetReadingStats(ctx, user2.ID)
	require.NoError(t, err)
	require.Equal(t, 0, stats.TotalTracked)
}

// ---- GetReadingStreak tests ----

func TestGetReadingStreak_NoData(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForReadingProgress(t, d, "streak-empty@example.com")

	streak, err := d.GetReadingStreak(t.Context(), user.ID)
	require.NoError(t, err)
	require.Equal(t, 0, streak)
}

func TestGetReadingStreak_TodayOnly(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForReadingProgress(t, d, "streak-today@example.com")
	ctx := t.Context()

	_, err := d.UpsertReadingProgress(ctx, user.ID, "doc-a", "/p[1]", 0.5, nil, nil)
	require.NoError(t, err)

	streak, err := d.GetReadingStreak(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 1, streak)
}

func TestGetReadingStreak_MultipleDocsToday(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForReadingProgress(t, d, "streak-multi@example.com")
	ctx := t.Context()

	_, err := d.UpsertReadingProgress(ctx, user.ID, "doc-a", "/p[1]", 0.5, nil, nil)
	require.NoError(t, err)
	_, err = d.UpsertReadingProgress(ctx, user.ID, "doc-b", "/p[2]", 0.3, nil, nil)
	require.NoError(t, err)

	// Multiple documents synced today should still yield streak = 1 (one day).
	streak, err := d.GetReadingStreak(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 1, streak)
}

func TestGetReadingStreak_IsolatedByUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUserForReadingProgress(t, d, "streak-u1@example.com")
	user2 := createTestUserForReadingProgress(t, d, "streak-u2@example.com")
	ctx := t.Context()

	_, err := d.UpsertReadingProgress(ctx, user1.ID, "doc-a", "/p[1]", 0.5, nil, nil)
	require.NoError(t, err)

	streak, err := d.GetReadingStreak(ctx, user2.ID)
	require.NoError(t, err)
	require.Equal(t, 0, streak)
}

func TestGetReadingStreak_ConsecutiveDays(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForReadingProgress(t, d, "streak-consec@example.com")
	ctx := t.Context()

	// Insert a document with an old timestamp, then update it through direct
	// SQL to simulate activity over consecutive days in the past, ending today.
	now := time.Now().UTC()

	// Create the initial entry.
	_, err := d.UpsertReadingProgress(ctx, user.ID, "book", "/p[1]", 0.1, nil, nil)
	require.NoError(t, err)

	// Backdate the created_at so we can set updated_at to simulate past days.
	// Update the row so updated_at lands on each of today, yesterday, 2 days ago.
	for offset := 0; offset <= 2; offset++ {
		day := now.AddDate(0, 0, -offset).Format("2006-01-02 15:04:05")
		// Each loop body uses the day-offset as the new "last synced" time.
		// We insert a new row for each simulated day using a unique doc name.
		doc := "book-day-" + day
		// Insert the row with updated_at pointing to the simulated day.
		_, err := d.ExecContext(ctx,
			`INSERT INTO reading_progress (user_id, document, progress, percentage, updated_at)
			 VALUES ($1, $2, '/p[1]', 0.2, $3)`,
			user.ID, doc, day,
		)
		require.NoError(t, err)
	}

	streak, err := d.GetReadingStreak(ctx, user.ID)
	require.NoError(t, err)
	// today (from UpsertReadingProgress) + 3 injected days (today, yesterday, 2 days ago)
	// but today is deduplicated → today + yesterday + 2-days-ago = 3 days total
	require.Equal(t, 3, streak)
}

func TestGetReadingStreak_GapBreaksStreak(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForReadingProgress(t, d, "streak-gap@example.com")
	ctx := t.Context()

	now := time.Now().UTC()

	// Insert activity today.
	_, err := d.UpsertReadingProgress(ctx, user.ID, "book-today", "/p[1]", 0.5, nil, nil)
	require.NoError(t, err)

	// Insert activity 3 days ago (gap: yesterday and 2 days ago are missing).
	threeDaysAgo := now.AddDate(0, 0, -3).Format("2006-01-02 15:04:05")
	_, err = d.ExecContext(ctx,
		`INSERT INTO reading_progress (user_id, document, progress, percentage, updated_at)
		 VALUES ($1, 'book-old', '/p[1]', 0.5, $2)`,
		user.ID, threeDaysAgo,
	)
	require.NoError(t, err)

	streak, err := d.GetReadingStreak(ctx, user.ID)
	require.NoError(t, err)
	// Only today counts; 3-days-ago breaks the chain.
	require.Equal(t, 1, streak)
}

func TestGetReadingStreak_OldActivityReturnsZero(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForReadingProgress(t, d, "streak-old@example.com")
	ctx := t.Context()

	// Insert activity that ended 5 days ago (before yesterday).
	fiveDaysAgo := time.Now().UTC().AddDate(0, 0, -5).Format("2006-01-02 15:04:05")
	_, err := d.ExecContext(ctx,
		`INSERT INTO reading_progress (user_id, document, progress, percentage, updated_at)
		 VALUES ($1, 'old-book', '/p[1]', 0.4, $2)`,
		user.ID, fiveDaysAgo,
	)
	require.NoError(t, err)

	streak, err := d.GetReadingStreak(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 0, streak)
}
