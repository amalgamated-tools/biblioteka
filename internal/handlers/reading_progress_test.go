package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

func setupReadingProgressHandler(t *testing.T) (*ReadingProgressHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &ReadingProgressHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Reader", "reader@example.com", "pw")
	require.NoError(t, err)
	return h, user.ID
}

func TestReadingProgressStats_MethodNotAllowed(t *testing.T) {
	h, userID := setupReadingProgressHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/reading-progress/stats", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleReadingProgressStats(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestReadingProgressStats_EmptyData(t *testing.T) {
	h, userID := setupReadingProgressHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-progress/stats", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleReadingProgressStats(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp readingProgressStatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.CurrentStreak)
	require.Equal(t, 0, resp.TotalTracked)
	require.Equal(t, 0, resp.TotalFinished)
	// in_progress must be [] not null in JSON
	require.NotNil(t, resp.InProgress)
	require.Empty(t, resp.InProgress)
}

func TestReadingProgressStats_InProgressItemsFiltered(t *testing.T) {
	h, userID := setupReadingProgressHandler(t)
	ctx := t.Context()

	// percentage = 0: not started — must not appear in in_progress
	_, err := h.DB.UpsertReadingProgress(ctx, userID, "doc-zero", "/p[1]", 0.0, nil, nil)
	require.NoError(t, err)

	// in-progress
	device := "Kindle"
	_, err = h.DB.UpsertReadingProgress(ctx, userID, "doc-mid", "/p[5]", 0.45, &device, nil)
	require.NoError(t, err)

	// finished (>= 0.99)
	_, err = h.DB.UpsertReadingProgress(ctx, userID, "doc-done", "/p[100]", 1.0, nil, nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-progress/stats", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleReadingProgressStats(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp readingProgressStatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 3, resp.TotalTracked)
	require.Equal(t, 1, resp.TotalFinished)
	require.Len(t, resp.InProgress, 1)
	require.Equal(t, "doc-mid", resp.InProgress[0].Document)
	require.Equal(t, 0.45, resp.InProgress[0].Percentage)
	require.NotNil(t, resp.InProgress[0].Device)
	require.Equal(t, "Kindle", *resp.InProgress[0].Device)
}

func TestReadingProgressStats_LastSyncedRFC3339(t *testing.T) {
	h, userID := setupReadingProgressHandler(t)
	ctx := t.Context()

	_, err := h.DB.UpsertReadingProgress(ctx, userID, "doc-mid", "/p[5]", 0.3, nil, nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-progress/stats", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleReadingProgressStats(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp readingProgressStatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.InProgress, 1)

	_, err = time.Parse(time.RFC3339, resp.InProgress[0].LastSynced)
	require.NoError(t, err, "last_synced should be RFC3339")
}

func TestReadingProgressStats_EstimateOmittedWhenInsufficientData(t *testing.T) {
	h, userID := setupReadingProgressHandler(t)
	ctx := t.Context()

	// A fresh upsert has created_at ≈ updated_at (< 5 min elapsed), so no estimate.
	_, err := h.DB.UpsertReadingProgress(ctx, userID, "doc-fresh", "/p[3]", 0.4, nil, nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-progress/stats", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleReadingProgressStats(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp readingProgressStatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.InProgress, 1)
	require.Nil(t, resp.InProgress[0].EstimatedMinutesRemaining,
		"estimate should be nil when elapsed < 5 minutes")
}

func TestReadingProgressStats_IsolatedByUser(t *testing.T) {
	h, userID := setupReadingProgressHandler(t)
	ctx := t.Context()

	other, err := h.DB.CreateUser(ctx, "Other", "other@example.com", "pw2")
	require.NoError(t, err)
	_, err = h.DB.UpsertReadingProgress(ctx, other.ID, "other-doc", "/p[1]", 0.5, nil, nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-progress/stats", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleReadingProgressStats(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp readingProgressStatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.TotalTracked)
	require.Empty(t, resp.InProgress)
}

func TestReadingProgressStats_StreakIncluded(t *testing.T) {
	h, userID := setupReadingProgressHandler(t)
	ctx := t.Context()

	_, err := h.DB.UpsertReadingProgress(ctx, userID, "doc-a", "/p[1]", 0.5, nil, nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-progress/stats", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleReadingProgressStats(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp readingProgressStatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 1, resp.CurrentStreak)
}

// ---- estimateMinutesRemaining unit tests ----

func makeReadingProgress(percentage float64, elapsed time.Duration) *db.ReadingProgress {
	now := time.Now()
	return &db.ReadingProgress{
		Percentage: percentage,
		CreatedAt:  db.Timestamp{Time: now.Add(-elapsed)},
		UpdatedAt:  db.Timestamp{Time: now},
	}
}

func TestEstimateMinutesRemaining_NilWhenPercentageTooLow(t *testing.T) {
	p := makeReadingProgress(0.005, 30*time.Minute)
	require.Nil(t, estimateMinutesRemaining(p))
}

func TestEstimateMinutesRemaining_NilWhenElapsedTooShort(t *testing.T) {
	p := makeReadingProgress(0.5, 3*time.Minute)
	require.Nil(t, estimateMinutesRemaining(p))
}

func TestEstimateMinutesRemaining_ComputedCorrectly(t *testing.T) {
	// 50% read in 60 minutes → ~60 minutes remaining.
	p := makeReadingProgress(0.5, 60*time.Minute)
	est := estimateMinutesRemaining(p)
	require.NotNil(t, est)
	require.Equal(t, int64(60), *est)
}

func TestEstimateMinutesRemaining_AlmostDone(t *testing.T) {
	// 90% read in 90 minutes → ~10 minutes remaining.
	p := makeReadingProgress(0.9, 90*time.Minute)
	est := estimateMinutesRemaining(p)
	require.NotNil(t, est)
	require.Equal(t, int64(10), *est)
}
