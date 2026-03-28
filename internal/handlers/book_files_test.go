package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupBookFileHandler(t *testing.T) (*BookFileHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &BookFileHandler{DB: d}
	user, err := d.CreateUser(context.Background(), "Test User", "test@example.com", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return h, user.ID
}

func TestGetBookFile_Handler(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	book, err := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	bf, err := h.DB.CreateBookFile(context.Background(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/"+bf.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var dto bookFileDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dto.FileName != "gunslinger.epub" {
		t.Errorf("file_name = %q, want %q", dto.FileName, "gunslinger.epub")
	}
}

func TestGetBookFile_NotFound(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteBookFile_Handler(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	book, err := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	bf, err := h.DB.CreateBookFile(context.Background(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}

	r := httptest.NewRequest(http.MethodDelete, "/api/book-files/"+bf.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
