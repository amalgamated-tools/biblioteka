package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTagHandler(t *testing.T) (*TagHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &TagHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, user.ID
}

func TestListTags_Handler(t *testing.T) {
	h, userID := setupTagHandler(t)

	// Create a tag first.
	body := mustMarshal(t, tagRequest{Name: "fiction"})
	r := httptest.NewRequest(http.MethodPost, "/api/tags", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleTags(w, r)
	require.Equal(t, http.StatusCreated, w.Code)

	// List tags.
	r = httptest.NewRequest(http.MethodGet, "/api/tags", nil)
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleTags(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var tags []tagDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tags))
	require.Len(t, tags, 1)
	require.Equal(t, "fiction", tags[0].Name)
}

func TestCreateTag_Handler(t *testing.T) {
	h, userID := setupTagHandler(t)

	body := mustMarshal(t, tagRequest{Name: "sci-fi"})
	r := httptest.NewRequest(http.MethodPost, "/api/tags", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleTags(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	var dto tagDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "sci-fi", dto.Name)
	require.NotEmpty(t, dto.ID)
}

func TestCreateTag_MissingName(t *testing.T) {
	h, userID := setupTagHandler(t)

	body := mustMarshal(t, tagRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/tags", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleTags(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateTag_Duplicate(t *testing.T) {
	h, userID := setupTagHandler(t)

	body := mustMarshal(t, tagRequest{Name: "fiction"})

	// Create first.
	r := httptest.NewRequest(http.MethodPost, "/api/tags", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleTags(w, r)
	require.Equal(t, http.StatusCreated, w.Code)

	// Create duplicate.
	body = mustMarshal(t, tagRequest{Name: "fiction"})
	r = httptest.NewRequest(http.MethodPost, "/api/tags", bytes.NewReader(body))
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleTags(w, r)

	require.Equal(t, http.StatusConflict, w.Code)
}

func TestGetTag_Handler(t *testing.T) {
	h, userID := setupTagHandler(t)

	// Create a tag.
	tag, err := h.DB.CreateTag(t.Context(), "mystery")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/tags/"+tag.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleTag(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dto tagDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "mystery", dto.Name)
}

func TestGetTag_NotFound(t *testing.T) {
	h, userID := setupTagHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/tags/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleTag(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateTag_Handler(t *testing.T) {
	h, userID := setupTagHandler(t)

	tag, err := h.DB.CreateTag(t.Context(), "old-name")
	require.NoError(t, err)

	body := mustMarshal(t, tagRequest{Name: "new-name"})
	r := httptest.NewRequest(http.MethodPut, "/api/tags/"+tag.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleTag(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dto tagDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "new-name", dto.Name)
}

func TestDeleteTag_Handler(t *testing.T) {
	h, userID := setupTagHandler(t)

	tag, err := h.DB.CreateTag(t.Context(), "to-delete")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodDelete, "/api/tags/"+tag.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleTag(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)

	// Verify it's gone.
	r = httptest.NewRequest(http.MethodGet, "/api/tags/"+tag.ID, nil)
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleTag(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleTags_MethodNotAllowed(t *testing.T) {
	h, userID := setupTagHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/tags", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleTags(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
