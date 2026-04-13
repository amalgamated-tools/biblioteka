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

func setupStatsHandler(t *testing.T) (*StatsHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &StatsHandler{DB: d}
	u, err := d.CreateUser(t.Context(), "Stats User", "stats@example.com", "secret")
	require.NoError(t, err, "create user")
	return h, u.ID
}

func TestHandleDownloadsPerMonth_Default(t *testing.T) {
	h, userID := setupStatsHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/stats/downloads-per-month", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleDownloadsPerMonth(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []monthlyDownloadsDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, defaultStatMonths, "default 12 months")
	for _, d := range dtos {
		require.Equal(t, 0, d.Count)
	}
}

func TestHandleDownloadsPerMonth_CustomMonths(t *testing.T) {
	h, userID := setupStatsHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/stats/downloads-per-month?months=3", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleDownloadsPerMonth(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []monthlyDownloadsDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, 3)
}

func TestHandleDownloadsPerMonth_ClampsMax(t *testing.T) {
	h, userID := setupStatsHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/stats/downloads-per-month?months=999", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleDownloadsPerMonth(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []monthlyDownloadsDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, maxStatMonths, "should be clamped to maxStatMonths")
}

func TestHandleDownloadsPerMonth_WithDownloads(t *testing.T) {
	h, userID := setupStatsHandler(t)

	// Create a book file and record two downloads.
	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Stats Book"})
	require.NoError(t, err)
	bf, err := h.DB.CreateBookFile(t.Context(), book.ID, "epub", "stats.epub", 1024, nil, "/books/stats.epub")
	require.NoError(t, err)

	require.NoError(t, h.DB.RecordBookDownload(t.Context(), bf.ID, userID))
	require.NoError(t, h.DB.RecordBookDownload(t.Context(), bf.ID, userID))

	r := httptest.NewRequest(http.MethodGet, "/api/stats/downloads-per-month?months=1", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleDownloadsPerMonth(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []monthlyDownloadsDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, 1)

	thisMonth := time.Now().UTC().Format("2006-01")
	require.Equal(t, thisMonth, dtos[0].Month)
	require.Equal(t, 2, dtos[0].Count)
}

func TestHandleDownloadsPerMonth_MethodNotAllowed(t *testing.T) {
	h, userID := setupStatsHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/stats/downloads-per-month", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleDownloadsPerMonth(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
