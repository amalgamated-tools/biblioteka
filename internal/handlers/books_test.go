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
		t.Fatalf("create user: %v", err)
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
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var dto bookDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
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

	body, _ := json.Marshal(bookRequest{})
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

	h.DB.CreateBook(context.Background(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var dtos []bookSummaryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(dtos) != 2 {
		t.Errorf("len = %d, want 2", len(dtos))
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

	b, _ := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodDelete, "/api/books/"+b.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
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
	if err := json.Unmarshal(w2.Body.Bytes(), &authors); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(authors) != 1 {
		t.Fatalf("len = %d, want 1", len(authors))
	}
	if authors[0].Name != "Stephen King" {
		t.Errorf("name = %q, want %q", authors[0].Name, "Stephen King")
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
	if err := json.Unmarshal(w2.Body.Bytes(), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len = %d, want 1", len(entries))
	}
	if entries[0].Series.Name != "The Dark Tower" {
		t.Errorf("series name = %q, want %q", entries[0].Series.Name, "The Dark Tower")
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
	if err := json.Unmarshal(w2.Body.Bytes(), &files); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("len = %d, want 1", len(files))
	}
}
