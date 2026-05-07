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

func TestGetBookTags_Empty(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Test Book"})
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/tags", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []tagDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 0)
}

func TestGetBookTags_WithTags(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Test Book"})
	require.NoError(t, err)
	tag1, err := h.DB.CreateTag(t.Context(), "fiction")
	require.NoError(t, err)
	tag2, err := h.DB.CreateTag(t.Context(), "sci-fi")
	require.NoError(t, err)
	require.NoError(t, h.DB.SetBookTags(t.Context(), b.ID, []string{tag1.ID, tag2.ID}))

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/tags", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []tagDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 2)
}

func TestPutBookTags_Success(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Test Book"})
	require.NoError(t, err)
	tag, err := h.DB.CreateTag(t.Context(), "mystery")
	require.NoError(t, err)

	body := mustMarshal(t, setBookTagsRequest{TagIDs: []string{tag.ID}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/tags", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []tagDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 1)
	require.Equal(t, "mystery", dtos[0].Name)
}

func TestPutBookTags_ClearsExisting(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Test Book"})
	require.NoError(t, err)
	tag1, err := h.DB.CreateTag(t.Context(), "tag-one")
	require.NoError(t, err)
	tag2, err := h.DB.CreateTag(t.Context(), "tag-two")
	require.NoError(t, err)

	// Set two tags initially.
	body := mustMarshal(t, setBookTagsRequest{TagIDs: []string{tag1.ID, tag2.ID}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/tags", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleBookRoutes(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	// Replace with just one tag.
	body2 := mustMarshal(t, setBookTagsRequest{TagIDs: []string{tag1.ID}})
	r2 := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/tags", bytes.NewReader(body2))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleBookRoutes(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code)

	var dtos []tagDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &dtos))
	require.Len(t, dtos, 1)
}

func TestPutBookTags_InvalidJSON(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Test Book"})
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/tags", strings.NewReader("not-json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPutBookTags_EmptyList(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Test Book"})
	require.NoError(t, err)

	body := mustMarshal(t, setBookTagsRequest{TagIDs: []string{}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/tags", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []tagDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 0)
}
