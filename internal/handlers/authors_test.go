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

	require.Equal(t, http.StatusCreated, w.Code)

	var dto authorDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "Stephen King", dto.Name)
}

func TestCreateAuthor_MissingName(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	body := mustMarshal(t, authorRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthors(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAuthor_WhitespaceOnlyName(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	body := mustMarshal(t, authorRequest{Name: "   "})
	r := httptest.NewRequest(http.MethodPost, "/api/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthors(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
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

	require.Equal(t, http.StatusBadRequest, w.Code)
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

	require.Equal(t, http.StatusConflict, w2.Code)
}

func TestListAuthors_Handler(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	_, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")
	_, err = h.DB.CreateAuthor(t.Context(), "Brandon Sanderson", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	r := httptest.NewRequest(http.MethodGet, "/api/authors", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthors(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []authorDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, 2)
}

func TestGetAuthor_Handler(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	a, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	r := httptest.NewRequest(http.MethodGet, "/api/authors/"+a.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetAuthor_NotFound(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/authors/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteAuthor_Handler(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	a, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	r := httptest.NewRequest(http.MethodDelete, "/api/authors/"+a.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteAuthor_NotFound(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/authors/nonexistent-id", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "author not found", resp.Error)
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

	require.Equal(t, http.StatusOK, w.Code)

	var result bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "unmarshal")
	require.Equal(t, 2, result.Total)
	require.Len(t, result.Books, 2)
}

func TestListAuthorBooks_AuthorNotFound(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/authors/nonexistent/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListAuthorBooks_Empty(t *testing.T) {
	h, userID := setupAuthorHandler(t)

	a, err := h.DB.CreateAuthor(t.Context(), "Unknown Author", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	r := httptest.NewRequest(http.MethodGet, "/api/authors/"+a.ID+"/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthor(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var result bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "unmarshal")
	require.Equal(t, 0, result.Total)
	require.NotNil(t, result.Books)
}
