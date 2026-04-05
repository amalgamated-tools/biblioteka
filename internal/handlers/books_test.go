package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

func setupBookHandler(t *testing.T) (*BookHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &BookHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, user.ID
}

func TestCreateBook_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	body := mustMarshal(t, bookRequest{Title: "The Gunslinger"})
	r := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var dto bookDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	if dto.Title != "The Gunslinger" {
		t.Errorf("title = %q, want %q", dto.Title, "The Gunslinger")
	}
	if dto.Authors == nil {
		t.Error("authors should be empty array, not nil")
	}
	if dto.Series == nil {
		t.Error("series should be empty array, not nil")
	}
	if dto.Files == nil {
		t.Error("files should be empty array, not nil")
	}
}

func TestCreateBook_MissingTitle(t *testing.T) {
	h, userID := setupBookHandler(t)

	body := mustMarshal(t, bookRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListBooks_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	if _, err := h.DB.CreateBook(t.Context(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}
	if _, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	r := httptest.NewRequest(http.MethodGet, "/api/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if len(resp.Books) != 2 {
		t.Errorf("len = %d, want 2", len(resp.Books))
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
	if resp.Limit != 50 {
		t.Errorf("limit = %d, want 50", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("offset = %d, want 0", resp.Offset)
	}
}

func TestListBooks_InvalidLimitOffset_NonInt(t *testing.T) {
	h, userID := setupBookHandler(t)

	// Seed some data
	if _, err := h.DB.CreateBook(t.Context(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}
	if _, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	// Provide non-integer limit/offset; handler should fall back to defaults.
	r := httptest.NewRequest(http.MethodGet, "/api/books?limit=abc&offset=xyz", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")

	if len(resp.Books) != 2 {
		t.Errorf("len = %d, want 2", len(resp.Books))
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
	// Invalid values should cause defaults to be used.
	if resp.Limit != 50 {
		t.Errorf("limit = %d, want 50 (default)", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("offset = %d, want 0 (default)", resp.Offset)
	}
}

func TestListBooks_NegativeLimitOffset(t *testing.T) {
	h, userID := setupBookHandler(t)

	if _, err := h.DB.CreateBook(t.Context(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}
	if _, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	r := httptest.NewRequest(http.MethodGet, "/api/books?limit=-5&offset=-10", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")

	if len(resp.Books) != 2 {
		t.Errorf("len = %d, want 2", len(resp.Books))
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}

	// Negative values should be clamped to safe non-negative values.
	if resp.Limit <= 0 {
		t.Errorf("limit = %d, want > 0 after clamping", resp.Limit)
	}
	if resp.Offset < 0 {
		t.Errorf("offset = %d, want >= 0 after clamping", resp.Offset)
	}
	if resp.Limit == -5 {
		t.Errorf("limit should not echo negative input; got %d", resp.Limit)
	}
	if resp.Offset == -10 {
		t.Errorf("offset should not echo negative input; got %d", resp.Offset)
	}
}

func TestListBooks_MaxLimitClamping(t *testing.T) {
	h, userID := setupBookHandler(t)

	if _, err := h.DB.CreateBook(t.Context(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}
	if _, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	// Request an absurdly large limit; handler should clamp to a maximum.
	r := httptest.NewRequest(http.MethodGet, "/api/books?limit=999999&offset=0", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")

	if len(resp.Books) != 2 {
		t.Errorf("len = %d, want 2", len(resp.Books))
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}

	// We don't assert the exact max, only that the requested huge limit was clamped.
	if resp.Limit == 999999 {
		t.Errorf("limit should be clamped below requested huge value; got %d", resp.Limit)
	}
	if resp.Limit < len(resp.Books) {
		t.Errorf("limit = %d, want >= number of returned books (%d)", resp.Limit, len(resp.Books))
	}
}

func TestListBooks_Search_MatchesTitle(t *testing.T) {
	h, userID := setupBookHandler(t)

	if _, err := h.DB.CreateBook(t.Context(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}
	if _, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	r := httptest.NewRequest(http.MethodGet, "/api/books?query=Gunslinger", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if len(resp.Books) != 1 {
		t.Errorf("len = %d, want 1", len(resp.Books))
	}
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
	if len(resp.Books) > 0 && resp.Books[0].Title != "The Gunslinger" {
		t.Errorf("title = %q, want %q", resp.Books[0].Title, "The Gunslinger")
	}
}

func TestListBooks_Search_NoResults(t *testing.T) {
	h, userID := setupBookHandler(t)

	if _, err := h.DB.CreateBook(t.Context(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	r := httptest.NewRequest(http.MethodGet, "/api/books?query=nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if len(resp.Books) != 0 {
		t.Errorf("len = %d, want 0", len(resp.Books))
	}
	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
}

func TestListBooks_EmptyQuery_ReturnsAll(t *testing.T) {
	h, userID := setupBookHandler(t)

	if _, err := h.DB.CreateBook(t.Context(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}
	if _, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	// Empty query string should behave like no query (list all).
	r := httptest.NewRequest(http.MethodGet, "/api/books?query=", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if len(resp.Books) != 2 {
		t.Errorf("len = %d, want 2", len(resp.Books))
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
}

func TestListBooks_WhitespaceOnlyQuery_ReturnsAll(t *testing.T) {
	h, userID := setupBookHandler(t)

	if _, err := h.DB.CreateBook(t.Context(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}
	if _, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	// Whitespace-only query should behave like no query (list all).
	r := httptest.NewRequest(http.MethodGet, "/api/books?query=%20%20%20", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if len(resp.Books) != 2 {
		t.Errorf("len = %d, want 2", len(resp.Books))
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
}

func TestGetBook_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestGetBook_NotFound(t *testing.T) {
	h, userID := setupBookHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/books/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteBook_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodDelete, "/api/books/"+b.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestDeleteBook_NotFound(t *testing.T) {
	h, userID := setupBookHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/books/nonexistent-id", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.Error != "book not found" {
		t.Errorf("error = %q, want %q", resp.Error, "book not found")
	}
}

func TestBookAuthors_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	a, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	// Set authors
	body := mustMarshal(t, map[string][]string{"author_ids": {a.ID}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("PUT status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Get authors
	r2 := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/authors", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()

	h.HandleBookRoutes(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("GET status = %d, want %d", w2.Code, http.StatusOK)
	}

	var authors []authorDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &authors), "unmarshal")
	require.Len(t, authors, 1)
	if authors[0].Name != "Stephen King" {
		t.Errorf("name = %q, want %q", authors[0].Name, "Stephen King")
	}
}

func TestBookSeries_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")

	pos := 1.0
	body := mustMarshal(t, map[string][]db.BookSeriesInput{
		"entries": {{SeriesID: s.ID, Position: &pos}},
	})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("PUT status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Get series
	r2 := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/series", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()

	h.HandleBookRoutes(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("GET status = %d, want %d", w2.Code, http.StatusOK)
	}

	var entries []bookSeriesEntryDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &entries), "unmarshal")
	require.Len(t, entries, 1)
	if entries[0].Series.Name != "The Dark Tower" {
		t.Errorf("series name = %q, want %q", entries[0].Series.Name, "The Dark Tower")
	}
}

func TestBookFiles_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	// Create a file
	body := mustMarshal(t, map[string]any{
		"file_type": "epub",
		"file_name": "gunslinger.epub",
		"file_size": 1024,
		"file_path": "/books/gunslinger.epub",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("POST status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// List files
	r2 := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/files", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()

	h.HandleBookRoutes(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("GET status = %d, want %d", w2.Code, http.StatusOK)
	}

	var files []bookFileDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &files), "unmarshal")
	require.Len(t, files, 1)
}
