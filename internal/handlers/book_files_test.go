package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// mockEmailSender is a test double for email.Sender.
type mockEmailSender struct {
	err      error
	called   bool
	lastTo   string
	lastFile string
}

func (m *mockEmailSender) SendWithAttachment(_ context.Context, to, _, _, fileName string, _ []byte, _ string) error {
	m.called = true
	m.lastTo = to
	m.lastFile = fileName
	return m.err
}

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

	book, _ := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	bf, _ := h.DB.CreateBookFile(context.Background(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")

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

	book, _ := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	bf, _ := h.DB.CreateBookFile(context.Background(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")

	r := httptest.NewRequest(http.MethodDelete, "/api/book-files/"+bf.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestSendBookFile_Success(t *testing.T) {
	h, userID := setupBookFileHandler(t)
	sender := &mockEmailSender{}
	h.Emailer = sender

	// Write a temporary file so the handler can read it.
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "gunslinger.epub")
	if err := os.WriteFile(filePath, []byte("fake epub"), 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	book, _ := h.DB.CreateBook("The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	bf, _ := h.DB.CreateBookFile(book.ID, "epub", "gunslinger.epub", 9, nil, filePath)

	body, _ := json.Marshal(map[string]string{"email": "user@kindle.com"})
	r := httptest.NewRequest(http.MethodPost, "/api/book-files/"+bf.ID+"/send", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !sender.called {
		t.Error("expected email sender to be called")
	}
	if sender.lastTo != "user@kindle.com" {
		t.Errorf("sent to %q, want user@kindle.com", sender.lastTo)
	}
	if sender.lastFile != "gunslinger.epub" {
		t.Errorf("sent file %q, want gunslinger.epub", sender.lastFile)
	}
}

func TestSendBookFile_MissingEmail(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	book, _ := h.DB.CreateBook("The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	bf, _ := h.DB.CreateBookFile(book.ID, "epub", "gunslinger.epub", 9, nil, "/some/path.epub")

	body, _ := json.Marshal(map[string]string{"email": ""})
	r := httptest.NewRequest(http.MethodPost, "/api/book-files/"+bf.ID+"/send", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestSendBookFile_InvalidEmail(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	book, _ := h.DB.CreateBook("The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	bf, _ := h.DB.CreateBookFile(book.ID, "epub", "gunslinger.epub", 9, nil, "/some/path.epub")

	body, _ := json.Marshal(map[string]string{"email": "notanemail"})
	r := httptest.NewRequest(http.MethodPost, "/api/book-files/"+bf.ID+"/send", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestSendBookFile_NotFound(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	body, _ := json.Marshal(map[string]string{"email": "user@kindle.com"})
	r := httptest.NewRequest(http.MethodPost, "/api/book-files/nonexistent/send", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestSendBookFile_MethodNotAllowed(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	book, _ := h.DB.CreateBook("The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	bf, _ := h.DB.CreateBookFile(book.ID, "epub", "gunslinger.epub", 9, nil, "/some/path.epub")

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/"+bf.ID+"/send", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
