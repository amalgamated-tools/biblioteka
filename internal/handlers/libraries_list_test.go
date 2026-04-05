package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListLibraryBooks_Success(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	// Create a library.
	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{Name: "Fiction", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	require.Equal(t, http.StatusCreated, w.Code, "create library")
	var lib libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lib), "unmarshal library")

	// Create a book and link it to the library.
	book, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	require.NoError(t, h.DB.AddBookToLibrary(t.Context(), lib.ID, book.ID), "add book to library")

	// List books for the library.
	r2 := httptest.NewRequest(http.MethodGet, "/api/libraries/"+lib.ID+"/books", nil)
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()
	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp), "unmarshal books")
	require.Equal(t, 1, len(resp.Books), "books count")
	if resp.Books[0].Title != "The Gunslinger" {
		t.Errorf("title = %q, want %q", resp.Books[0].Title, "The Gunslinger")
	}
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
}

func TestListLibraryBooks_PaginationValid(t *testing.T) {
	h, userID, _ := setupLibraryHandler(t)

	// Create a library.
	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{Name: "Paginated", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	require.Equal(t, http.StatusCreated, w.Code, "create library")
	var lib libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lib), "unmarshal library")

	// Create multiple books and link them to the library.
	const totalBooks = 3
	for i := range totalBooks {
		title := fmt.Sprintf("Book %d", i+1)
		book, err := h.DB.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err, "create book %d", i+1)
		require.NoError(t, h.DB.AddBookToLibrary(t.Context(), lib.ID, book.ID), "add book %d to library", i+1)
	}

	// Request a paginated slice of books.
	r2 := httptest.NewRequest(http.MethodGet, "/api/libraries/"+lib.ID+"/books?limit=2&offset=1", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleLibrary(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code, "list books status")

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp), "unmarshal books")

	require.Equal(t, totalBooks, resp.Total, "total")
	require.True(t, len(resp.Books) >= 1 && len(resp.Books) <= 2, "books count = %d, want between 1 and 2", len(resp.Books))
}

func TestListLibraryBooks_PaginationInvalidValues(t *testing.T) {
	h, userID, _ := setupLibraryHandler(t)

	// Create a library.
	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{Name: "InvalidPagination", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	require.Equal(t, http.StatusCreated, w.Code, "create library")
	var lib libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lib), "unmarshal library")

	// Create multiple books and link them to the library.
	const totalBooks = 3
	for i := range totalBooks {
		title := fmt.Sprintf("Invalid Book %d", i+1)
		book, err := h.DB.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err, "create book %d", i+1)
		require.NoError(t, h.DB.AddBookToLibrary(t.Context(), lib.ID, book.ID), "add book %d to library", i+1)
	}

	// Use negative values that should be validated/clamped by the handler.
	r2 := httptest.NewRequest(http.MethodGet, "/api/libraries/"+lib.ID+"/books?limit=-5&offset=-10", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleLibrary(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code, "list books status")

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp), "unmarshal books")

	require.Equal(t, totalBooks, resp.Total, "total")
	require.NotEmpty(t, resp.Books, "books count")
}

func TestListLibraryBooks_PaginationMaxLimitClamping(t *testing.T) {
	h, userID, _ := setupLibraryHandler(t)

	// Create a library.
	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{Name: "MaxLimit", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	require.Equal(t, http.StatusCreated, w.Code, "create library")
	var lib libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lib), "unmarshal library")

	// Create several books and link them to the library.
	const totalBooks = 10
	for i := range totalBooks {
		title := fmt.Sprintf("Clamped Book %d", i+1)
		book, err := h.DB.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err, "create book %d", i+1)
		require.NoError(t, h.DB.AddBookToLibrary(t.Context(), lib.ID, book.ID), "add book %d to library", i+1)
	}

	// Request with an excessively large limit to ensure it is clamped internally.
	r2 := httptest.NewRequest(http.MethodGet, "/api/libraries/"+lib.ID+"/books?limit=999999&offset=0", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleLibrary(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code, "list books status")

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp), "unmarshal books")

	require.Equal(t, totalBooks, resp.Total, "total")
	require.Equal(t, totalBooks, len(resp.Books), "books count")
}

func TestListLibraryBooks_NotFound(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/libraries/nonexistent-id/books", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibrary(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestListLibraryBooks_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	// Create a library first.
	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{Name: "Fiction", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	require.Equal(t, http.StatusCreated, w.Code, "create library")
	var lib libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lib), "unmarshal library")

	// POST to /books sub-resource should be method not allowed.
	r2 := httptest.NewRequest(http.MethodPost, "/api/libraries/"+lib.ID+"/books", nil)
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()
	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusMethodNotAllowed, w2.Body.String())
	}
}

func TestListLibraries_NonAdminAllowed(t *testing.T) {
	h, _, regularID := setupLibraryHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/libraries", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
