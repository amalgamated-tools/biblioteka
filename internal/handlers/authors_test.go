package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupAuthorHandler(t *testing.T) (*AuthorHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &AuthorHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err, "create user")
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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
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
	require.NoError(t, err, "create author")

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
		require.NoError(t, err, "create author")
	}
	if _, err := h.DB.CreateAuthor(t.Context(), "Brandon Sanderson", nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create author")
	}

	r := httptest.NewRequest(http.MethodGet, "/api/authors", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthors(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var dtos []authorDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	if len(dtos) != 2 {
		t.Errorf("len = %d, want 2", len(dtos))
	}
}

func TestGetAuthor_Handler(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	a, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

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
	require.NoError(t, err, "create author")

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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.Error != "author not found" {
		t.Errorf("error = %q, want %q", resp.Error, "author not found")
	}
}

func TestListAuthorBooks_Handler(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	a, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	b1, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	b2, err := h.DB.CreateBook(t.Context(), "The Drawing of the Three", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	require.NoError(t, h.DB.SetBookAuthors(t.Context(), b1.ID, []string{a.ID}), "set book authors")
	require.NoError(t, h.DB.SetBookAuthors(t.Context(), b2.ID, []string{a.ID}), "set book authors")

	r := httptest.NewRequest(http.MethodGet, "/api/authors/"+a.ID+"/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "unmarshal")
	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}
	if len(result.Books) != 2 {
		t.Errorf("len(books) = %d, want 2", len(result.Books))
	}
}

func TestListAuthorBooks_AuthorNotFound(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/authors/nonexistent/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestListAuthorBooks_Empty(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	a, err := h.DB.CreateAuthor(t.Context(), "Unknown Author", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	r := httptest.NewRequest(http.MethodGet, "/api/authors/"+a.ID+"/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "unmarshal")
	if result.Total != 0 {
		t.Errorf("total = %d, want 0", result.Total)
	}
	if result.Books == nil {
		t.Error("books should be empty slice, not nil")
	}
}
