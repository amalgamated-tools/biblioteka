package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// createTestDownloadFixtures creates a user, book and book file and returns their IDs.
func createTestDownloadFixtures(t *testing.T, d *DB) (userID, bookFileID string) {
	t.Helper()
	u, err := d.CreateUser(t.Context(), "Test User", "dl@example.com", "secret")
	require.NoError(t, err, "CreateUser")

	b, err := d.CreateBook(t.Context(), BookInput{Title: "Downloads Test Book"})
	require.NoError(t, err, "CreateBook")

	bf, err := d.CreateBookFile(t.Context(), b.ID, "epub", "test.epub", 1024, nil, "/books/test.epub")
	require.NoError(t, err, "CreateBookFile")

	return u.ID, bf.ID
}

func TestRecordBookDownload(t *testing.T) {
	d := newTestDB(t)
	userID, bfID := createTestDownloadFixtures(t, d)

	err := d.RecordBookDownload(t.Context(), bfID, userID)
	require.NoError(t, err, "RecordBookDownload() first call")

	err = d.RecordBookDownload(t.Context(), bfID, userID)
	require.NoError(t, err, "RecordBookDownload() second call")

	// Verify two rows were inserted.
	var count int
	row := d.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM book_downloads WHERE user_id = $1`, userID)
	require.NoError(t, row.Scan(&count), "scan count")
	require.Equal(t, 2, count, "expected 2 download records")
}

func TestGetMonthlyDownloads_Empty(t *testing.T) {
	d := newTestDB(t)
	userID, _ := createTestDownloadFixtures(t, d)

	counts, err := d.GetMonthlyDownloads(t.Context(), userID, 3)
	require.NoError(t, err, "GetMonthlyDownloads() error")
	require.Len(t, counts, 3, "expected 3 months even with no downloads")
	for _, c := range counts {
		require.Equal(t, 0, c.Count, "expected zero downloads for month %s", c.Month)
	}
}

func TestGetMonthlyDownloads_CurrentMonth(t *testing.T) {
	d := newTestDB(t)
	userID, bfID := createTestDownloadFixtures(t, d)

	require.NoError(t, d.RecordBookDownload(t.Context(), bfID, userID))
	require.NoError(t, d.RecordBookDownload(t.Context(), bfID, userID))

	counts, err := d.GetMonthlyDownloads(t.Context(), userID, 3)
	require.NoError(t, err, "GetMonthlyDownloads() error")
	require.Len(t, counts, 3)

	// The last element should be the current month with count == 2.
	thisMonth := time.Now().UTC().Format("2006-01")
	last := counts[len(counts)-1]
	require.Equal(t, thisMonth, last.Month)
	require.Equal(t, 2, last.Count)
}

func TestGetMonthlyDownloads_UserIsolation(t *testing.T) {
	d := newTestDB(t)
	userID, bfID := createTestDownloadFixtures(t, d)

	// Create a second user.
	u2, err := d.CreateUser(t.Context(), "Other User", "other@example.com", "secret2")
	require.NoError(t, err, "CreateUser other")

	// Record downloads for both users.
	require.NoError(t, d.RecordBookDownload(t.Context(), bfID, userID))
	require.NoError(t, d.RecordBookDownload(t.Context(), bfID, u2.ID))

	// Each user should only see their own download.
	counts1, err := d.GetMonthlyDownloads(t.Context(), userID, 1)
	require.NoError(t, err)
	require.Len(t, counts1, 1)
	require.Equal(t, 1, counts1[0].Count)

	counts2, err := d.GetMonthlyDownloads(t.Context(), u2.ID, 1)
	require.NoError(t, err)
	require.Len(t, counts2, 1)
	require.Equal(t, 1, counts2[0].Count)
}

func TestGetMonthlyDownloads_OrderedOldestFirst(t *testing.T) {
	d := newTestDB(t)
	userID, bfID := createTestDownloadFixtures(t, d)

	require.NoError(t, d.RecordBookDownload(t.Context(), bfID, userID))

	counts, err := d.GetMonthlyDownloads(t.Context(), userID, 6)
	require.NoError(t, err)
	require.Len(t, counts, 6)

	for i := 1; i < len(counts); i++ {
		require.True(t, counts[i].Month >= counts[i-1].Month,
			"months should be ordered oldest first: got %s before %s", counts[i-1].Month, counts[i].Month)
	}
}
