package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupAuthorHandler(t *testing.T) (*AuthorHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &AuthorHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return h, user.ID
}

func TestCreateAuthor_Handler(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	body := mustMarshal(t, authorRequest{Name: "Stephen King"})
	r := httptest.NewRequest(http.MethodPost, "/api/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthors(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var dto authorDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dto.Name != "Stephen King" {
		t.Errorf("name = %q, want %q", dto.Name, "Stephen King")
	}
}

func TestCreateAuthor_MissingName(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	body := mustMarshal(t, authorRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthors(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateAuthor_WhitespaceOnlyName(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	body := mustMarshal(t, authorRequest{Name: "   "})
	r := httptest.NewRequest(http.MethodPost, "/api/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthors(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateAuthor_WhitespaceOnlyName(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	a, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create author: %v", err)
	}

	body := mustMarshal(t, authorRequest{Name: "   "})
	r := httptest.NewRequest(http.MethodPut, "/api/authors/"+a.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateAuthor_Duplicate(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	body := mustMarshal(t, authorRequest{Name: "Stephen King"})
	r := httptest.NewRequest(http.MethodPost, "/api/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAuthors(w, r)

	r2 := httptest.NewRequest(http.MethodPost, "/api/authors", bytes.NewReader(body))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleAuthors(w2, r2)

	if w2.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusConflict)
	}
}

func TestListAuthors_Handler(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	if _, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil); err != nil {
		t.Fatalf("create author: %v", err)
	}
	if _, err := h.DB.CreateAuthor(t.Context(), "Brandon Sanderson", nil, nil, nil, nil); err != nil {
		t.Fatalf("create author: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/authors", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthors(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var dtos []authorDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(dtos) != 2 {
		t.Errorf("len = %d, want 2", len(dtos))
	}
}

func TestGetAuthor_Handler(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	a, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create author: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/authors/"+a.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestGetAuthor_NotFound(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/authors/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteAuthor_Handler(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	a, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create author: %v", err)
	}

	r := httptest.NewRequest(http.MethodDelete, "/api/authors/"+a.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestDeleteAuthor_NotFound(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/authors/nonexistent-id", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != "author not found" {
		t.Errorf("error = %q, want %q", resp.Error, "author not found")
	}
}
