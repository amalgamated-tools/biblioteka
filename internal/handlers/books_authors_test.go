package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetBookAuthors_Empty(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/authors", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var dtos []authorDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		require.NoError(t, err, "unmarshal")
	}
	if len(dtos) != 0 {
		t.Errorf("len = %d, want 0", len(dtos))
	}
}

func TestGetBookAuthors_WithAuthors(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	a1, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")
	a2, err := h.DB.CreateAuthor(t.Context(), "Another Author", nil, nil, nil, nil)
	require.NoError(t, err, "create author 2")
	if err := h.DB.SetBookAuthors(t.Context(), b.ID, []string{a1.ID, a2.ID}); err != nil {
		require.NoError(t, err, "set book authors")
	}

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/authors", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var dtos []authorDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		require.NoError(t, err, "unmarshal")
	}
	if len(dtos) != 2 {
		t.Errorf("len = %d, want 2", len(dtos))
	}
}

func TestPutBookAuthors_Success(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	a, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	body := mustMarshal(t, setBookAuthorsRequest{AuthorIDs: []string{a.ID}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var dtos []authorDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		require.NoError(t, err, "unmarshal")
	}
	if len(dtos) != 1 {
		require.Failf(t, "failed", "len = %d, want 1", len(dtos))
	}
	if dtos[0].Name != "Stephen King" {
		t.Errorf("name = %q, want %q", dtos[0].Name, "Stephen King")
	}
}

func TestPutBookAuthors_ClearsExisting(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "initial PUT status = %d; body: %s", w.Code, w.Body.String())
	}

	// Replace with just one author.
	body2 := mustMarshal(t, setBookAuthorsRequest{AuthorIDs: []string{a1.ID}})
	r2 := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", bytes.NewReader(body2))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleBookRoutes(w2, r2)

	if w2.Code != http.StatusOK {
		require.Failf(t, "failed", "replace PUT status = %d; body: %s", w2.Code, w2.Body.String())
	}

	var dtos []authorDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &dtos); err != nil {
		require.NoError(t, err, "unmarshal")
	}
	if len(dtos) != 1 {
		t.Errorf("len = %d, want 1 after replacement", len(dtos))
	}
}

func TestPutBookAuthors_InvalidJSON(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", strings.NewReader("not-json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBookAuthors_MethodNotAllowed(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/authors", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPutBookAuthors_EmptyList(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, setBookAuthorsRequest{AuthorIDs: []string{}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var dtos []authorDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		require.NoError(t, err, "unmarshal")
	}
	if len(dtos) != 0 {
		t.Errorf("len = %d, want 0", len(dtos))
	}
}
