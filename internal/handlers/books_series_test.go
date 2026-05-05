package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

func TestGetBookSeries_Empty(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/series", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var entries []bookSeriesEntryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &entries), "unmarshal")
	require.Len(t, entries, 0)
}

func TestGetBookSeries_WithEntries(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")

	pos := 1.0
	require.NoError(t, h.DB.SetBookSeries(t.Context(), b.ID, []db.BookSeriesInput{{SeriesID: s.ID, Position: &pos}}), "set book series")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/series", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var entries []bookSeriesEntryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &entries), "unmarshal")
	require.Len(t, entries, 1)
	require.Equal(t, "The Dark Tower", entries[0].Series.Name)
	require.NotNil(t, entries[0].Position)
	require.Equal(t, 1.0, *entries[0].Position)
}

func TestPutBookSeries_Success(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")

	pos := 1.0
	body := mustMarshal(t, setBookSeriesRequest{
		Entries: []db.BookSeriesInput{{SeriesID: s.ID, Position: &pos}},
	})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var entries []bookSeriesEntryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &entries), "unmarshal")
	require.Len(t, entries, 1)
	require.Equal(t, "The Dark Tower", entries[0].Series.Name)
}

func TestPutBookSeries_ClearsExisting(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	s1, err := h.DB.CreateSeries(t.Context(), "Series One", nil, nil, nil)
	require.NoError(t, err, "create series1")
	s2, err := h.DB.CreateSeries(t.Context(), "Series Two", nil, nil, nil)
	require.NoError(t, err, "create series2")

	pos1 := 1.0
	pos2 := 2.0

	// Set two series initially.
	body := mustMarshal(t, setBookSeriesRequest{
		Entries: []db.BookSeriesInput{
			{SeriesID: s1.ID, Position: &pos1},
			{SeriesID: s2.ID, Position: &pos2},
		},
	})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleBookRoutes(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	// Replace with just one series.
	body2 := mustMarshal(t, setBookSeriesRequest{
		Entries: []db.BookSeriesInput{{SeriesID: s1.ID, Position: &pos1}},
	})
	r2 := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body2))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleBookRoutes(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code)

	var entries []bookSeriesEntryDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &entries), "unmarshal")
	require.Len(t, entries, 1)
}

func TestPutBookSeries_InvalidJSON(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", strings.NewReader("not-json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBookSeries_MethodNotAllowed(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/series", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestPutBookSeries_EmptyEntries(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	body := mustMarshal(t, setBookSeriesRequest{Entries: []db.BookSeriesInput{}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var entries []bookSeriesEntryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &entries), "unmarshal")
	require.Len(t, entries, 0)
}

func TestPutBookSeries_AuditLog(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")

	pos := 1.0
	body := mustMarshal(t, setBookSeriesRequest{Entries: []db.BookSeriesInput{{SeriesID: s.ID, Position: &pos}}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err)

	var found bool
	for _, l := range logs {
		if l.Action == db.AuditActionBookSeriesUpdated && l.EntityID == b.ID {
			found = true
			break
		}
	}
	require.True(t, found, "expected a book.series_updated audit log entry")
}
