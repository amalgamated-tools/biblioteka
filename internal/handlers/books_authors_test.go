package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

func TestGetBookAuthors_Empty(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/authors", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []authorDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, 0)
}

func TestGetBookAuthors_WithAuthors(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	a1, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")
	a2, err := h.DB.CreateAuthor(t.Context(), "Another Author", nil, nil, nil, nil)
	require.NoError(t, err, "create author 2")
	require.NoError(t, h.DB.SetBookAuthors(t.Context(), b.ID, []string{a1.ID, a2.ID}), "set book authors")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/authors", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []authorDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, 2)
}

func TestPutBookAuthors_Success(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	a, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	body := mustMarshal(t, setBookAuthorsRequest{AuthorIDs: []string{a.ID}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []authorDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, 1)
	require.Equal(t, "Stephen King", dtos[0].Name)
}

func TestPutBookAuthors_ClearsExisting(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	a1, err := h.DB.CreateAuthor(t.Context(), "Author One", nil, nil, nil, nil)
	require.NoError(t, err, "create author1")
	a2, err := h.DB.CreateAuthor(t.Context(), "Author Two", nil, nil, nil, nil)
	require.NoError(t, err, "create author2")

	// Set two authors initially.
	body := mustMarshal(t, setBookAuthorsRequest{AuthorIDs: []string{a1.ID, a2.ID}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleBookRoutes(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	// Replace with just one author.
	body2 := mustMarshal(t, setBookAuthorsRequest{AuthorIDs: []string{a1.ID}})
	r2 := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", bytes.NewReader(body2))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleBookRoutes(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code)

	var dtos []authorDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, 1)
}

func TestPutBookAuthors_InvalidJSON(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", strings.NewReader("not-json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBookAuthors_MethodNotAllowed(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/authors", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestPutBookAuthors_EmptyList(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	body := mustMarshal(t, setBookAuthorsRequest{AuthorIDs: []string{}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []authorDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, 0)
}
