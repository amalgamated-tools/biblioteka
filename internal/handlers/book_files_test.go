package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

func setupBookFileHandler(t *testing.T) (*BookFileHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &BookFileHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, user.ID
}

func TestGetBookFile_Handler(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	bf, err := h.DB.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/"+bf.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dto bookFileDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "gunslinger.epub", dto.FileName)
}

func TestGetBookFile_NotFound(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteBookFile_Handler(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	bf, err := h.DB.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodDelete, "/api/book-files/"+bf.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}
