package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

func setupBookHandler(t *testing.T) (*BookHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &BookHandler{DB: d}
	user, err := d.CreateUser(context.Background(), "Test User", "test@example.com", "password1")
	if err != nil {
		failNowf(t, "create user: %v", err)
	}
	return h, user.ID
}

func TestCreateBook_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	body, _ := json.Marshal(bookRequest{Title: "The Gunslinger"})
	r := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusCreated {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var dto bookDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}
	if dto.Title != "The Gunslinger" {
		failf(t, "title = %q, want %q", dto.Title, "The Gunslinger")
	}
	if dto.Authors == nil {
		fail(t, "authors should be empty array, not nil")
	}
	if dto.Series == nil {
		fail(t, "series should be empty array, not nil")
	}
	if dto.Files == nil {
		fail(t, "files should be empty array, not nil")
	}
}

func TestCreateBook_MissingTitle(t *testing.T) {
	h, userID := setupBookHandler(t)

	body, _ := json.Marshal(bookRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusBadRequest {
		failf(t, "status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListBooks_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	if _, err := h.DB.CreateBook(context.Background(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		failNowf(t, "create book: %v", err)
	}
	if _, err := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		failNowf(t, "create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusOK {
		failf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp bookListDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}
	if len(resp.Books) != 2 {
		failf(t, "len = %d, want 2", len(resp.Books))
	}
	if resp.Total != 2 {
		failf(t, "total = %d, want 2", resp.Total)
	}
	if resp.Limit != 50 {
		failf(t, "limit = %d, want 50", resp.Limit)
	}
	if resp.Offset != 0 {
		failf(t, "offset = %d, want 0", resp.Offset)
	}
}

func TestListBooks_InvalidLimitOffset_NonInt(t *testing.T) {
	h, userID := setupBookHandler(t)

	// Seed some data
	if _, err := h.DB.CreateBook(context.Background(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		failNowf(t, "create book: %v", err)
	}
	if _, err := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		failNowf(t, "create book: %v", err)
	}

	// Provide non-integer limit/offset; handler should fall back to defaults.
	r := httptest.NewRequest(http.MethodGet, "/api/books?limit=abc&offset=xyz", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp bookListDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}

	if len(resp.Books) != 2 {
		failf(t, "len = %d, want 2", len(resp.Books))
	}
	if resp.Total != 2 {
		failf(t, "total = %d, want 2", resp.Total)
	}
	// Invalid values should cause defaults to be used.
	if resp.Limit != 50 {
		failf(t, "limit = %d, want 50 (default)", resp.Limit)
	}
	if resp.Offset != 0 {
		failf(t, "offset = %d, want 0 (default)", resp.Offset)
	}
}

func TestListBooks_NegativeLimitOffset(t *testing.T) {
	h, userID := setupBookHandler(t)

	if _, err := h.DB.CreateBook(context.Background(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		failNowf(t, "create book: %v", err)
	}
	if _, err := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		failNowf(t, "create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/books?limit=-5&offset=-10", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp bookListDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}

	if len(resp.Books) != 2 {
		failf(t, "len = %d, want 2", len(resp.Books))
	}
	if resp.Total != 2 {
		failf(t, "total = %d, want 2", resp.Total)
	}

	// Negative values should be clamped to safe non-negative values.
	if resp.Limit <= 0 {
		failf(t, "limit = %d, want > 0 after clamping", resp.Limit)
	}
	if resp.Offset < 0 {
		failf(t, "offset = %d, want >= 0 after clamping", resp.Offset)
	}
	if resp.Limit == -5 {
		failf(t, "limit should not echo negative input; got %d", resp.Limit)
	}
	if resp.Offset == -10 {
		failf(t, "offset should not echo negative input; got %d", resp.Offset)
	}
}

func TestListBooks_MaxLimitClamping(t *testing.T) {
	h, userID := setupBookHandler(t)

	if _, err := h.DB.CreateBook(context.Background(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		failNowf(t, "create book: %v", err)
	}
	if _, err := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		failNowf(t, "create book: %v", err)
	}

	// Request an absurdly large limit; handler should clamp to a maximum.
	r := httptest.NewRequest(http.MethodGet, "/api/books?limit=999999&offset=0", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp bookListDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}

	if len(resp.Books) != 2 {
		failf(t, "len = %d, want 2", len(resp.Books))
	}
	if resp.Total != 2 {
		failf(t, "total = %d, want 2", resp.Total)
	}

	// We don't assert the exact max, only that the requested huge limit was clamped.
	if resp.Limit == 999999 {
		failf(t, "limit should be clamped below requested huge value; got %d", resp.Limit)
	}
	if resp.Limit < len(resp.Books) {
		failf(t, "limit = %d, want >= number of returned books (%d)", resp.Limit, len(resp.Books))
	}
}

func TestGetBook_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, _ := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestGetBook_NotFound(t *testing.T) {
	h, userID := setupBookHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/books/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusNotFound {
		failf(t, "status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteBook_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, _ := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodDelete, "/api/books/"+b.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusNoContent {
		failf(t, "status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestBookAuthors_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, _ := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	a, _ := h.DB.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)

	// Set authors
	body, _ := json.Marshal(map[string][]string{"author_ids": {a.ID}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		failf(t, "PUT status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Get authors
	r2 := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/authors", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()

	h.HandleBookRoutes(w2, r2)

	if w2.Code != http.StatusOK {
		failf(t, "GET status = %d, want %d", w2.Code, http.StatusOK)
	}

	var authors []authorDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &authors); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}
	if len(authors) != 1 {
		failNowf(t, "len = %d, want 1", len(authors))
	}
	if authors[0].Name != "Stephen King" {
		failf(t, "name = %q, want %q", authors[0].Name, "Stephen King")
	}
}

func TestBookSeries_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, _ := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	s, _ := h.DB.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	pos := 1.0
	body, _ := json.Marshal(map[string][]db.BookSeriesInput{
		"entries": {{SeriesID: s.ID, Position: &pos}},
	})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		failf(t, "PUT status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Get series
	r2 := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/series", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()

	h.HandleBookRoutes(w2, r2)

	if w2.Code != http.StatusOK {
		failf(t, "GET status = %d, want %d", w2.Code, http.StatusOK)
	}

	var entries []bookSeriesEntryDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &entries); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}
	if len(entries) != 1 {
		failNowf(t, "len = %d, want 1", len(entries))
	}
	if entries[0].Series.Name != "The Dark Tower" {
		failf(t, "series name = %q, want %q", entries[0].Series.Name, "The Dark Tower")
	}
}

func TestBookFiles_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, _ := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Create a file
	body, _ := json.Marshal(map[string]any{
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
		failf(t, "POST status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// List files
	r2 := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/files", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()

	h.HandleBookRoutes(w2, r2)

	if w2.Code != http.StatusOK {
		failf(t, "GET status = %d, want %d", w2.Code, http.StatusOK)
	}

	var files []bookFileDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &files); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}
	if len(files) != 1 {
		failNowf(t, "len = %d, want 1", len(files))
	}
}
