package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

func setupMetadataHandler(t *testing.T) (*MetadataHandler, *BookHandler, string) {
	t.Helper()
	d := newTestDB(t)
	enq := &mockEnqueuer{}
	h := &MetadataHandler{DB: d, Enqueuer: enq}
	bh := &BookHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, bh, user.ID
}

func createTestBook(t *testing.T, d *db.DB, title string) *db.Book {
	t.Helper()
	book, err := d.CreateBook(t.Context(), db.BookInput{Title: title})
	require.NoError(t, err)
	return book
}

func TestGetPendingMetadata_Found(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	title := "Remote Title"
	_, err := h.DB.CreateGoodreadsMetadata(t.Context(), userID,
		db.GoodreadsMetadataInput{BookID: &book.ID, Title: &title},
	)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+book.ID+"/metadata", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusOK, w.Code)

	var dto metadataDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "goodreads", dto.Source)
	require.NotNil(t, dto.Title)
	require.Equal(t, "Remote Title", *dto.Title)
}

func TestGetPendingMetadata_NotFound(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+book.ID+"/metadata", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestFetchMetadata_Enqueues(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/fetch", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusAccepted, w.Code)

	var resp fetchMetadataResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "mock-job-id", resp.TaskID)

	enq := h.Enqueuer.(*mockEnqueuer)
	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Len(t, enq.jobs, 1)
	require.Equal(t, "enrich:goodreads", enq.jobs[0].Name)
}

func TestFetchMetadata_BookNotFound(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/books/nonexistent/metadata/fetch", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, "nonexistent")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestFetchMetadata_NoEnqueuer(t *testing.T) {
	d := newTestDB(t)
	h := &MetadataHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err)
	book := createTestBook(t, d, "Test Book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/fetch", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestFetchMetadata_EnqueueError(t *testing.T) {
	d := newTestDB(t)
	enq := &mockEnqueuer{err: fmt.Errorf("redis unavailable")}
	h := &MetadataHandler{DB: d, Enqueuer: enq}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err)
	book := createTestBook(t, d, "Test Book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/fetch", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestApplyMetadata(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Original Title")

	remoteTitle := "Better Title"
	remoteISBN := "9780593135204"
	_, err := h.DB.CreateGoodreadsMetadata(t.Context(), userID,
		db.GoodreadsMetadataInput{
			BookID: &book.ID,
			Title:  &remoteTitle,
			ISBN13: &remoteISBN,
		},
	)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/apply", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusOK, w.Code)

	var dto bookSummaryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "Better Title", dto.Title)
	require.NotNil(t, dto.ISBN13)
	require.Equal(t, "9780593135204", *dto.ISBN13)

	// Verify the book was updated in the database.
	updated, err := h.DB.GetBook(t.Context(), book.ID)
	require.NoError(t, err)
	require.Equal(t, "Better Title", updated.Title)

	// Verify the metadata was marked as applied.
	_, err = h.DB.GetPendingGoodreadsMetadataByBook(t.Context(), userID, book.ID)
	require.Error(t, err, "pending metadata should no longer exist after apply")
}

func TestApplyMetadata_NoPending(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/apply", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestRejectMetadata(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	title := "Remote Title"
	_, err := h.DB.CreateGoodreadsMetadata(t.Context(), userID,
		db.GoodreadsMetadataInput{BookID: &book.ID, Title: &title},
	)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/reject", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusNoContent, w.Code)

	// Verify the metadata was rejected (no longer pending).
	_, err = h.DB.GetPendingGoodreadsMetadataByBook(t.Context(), userID, book.ID)
	require.Error(t, err, "pending metadata should no longer exist after reject")
}

func TestRejectMetadata_NoPending(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/reject", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestMetadata_MethodNotAllowed(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"GET fetch", http.MethodGet, "/api/books/" + book.ID + "/metadata/fetch"},
		{"POST get", http.MethodPost, "/api/books/" + book.ID + "/metadata"},
		{"GET apply", http.MethodGet, "/api/books/" + book.ID + "/metadata/apply"},
		{"GET reject", http.MethodGet, "/api/books/" + book.ID + "/metadata/reject"},
		{"POST events", http.MethodPost, "/api/books/" + book.ID + "/metadata/events"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, nil)
			r = withUserID(r, userID)
			w := httptest.NewRecorder()

			h.HandleBookMetadata(w, r, book.ID)
			require.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

func TestMetadata_UnknownSubPath(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+book.ID+"/metadata/unknown", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusNotFound, w.Code)
}
