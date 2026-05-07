package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

func setupReadingListHandler(t *testing.T) (*ReadingListHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &ReadingListHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, user.ID
}

// --- List / Create ---

func TestHandleReadingLists_List(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	_, err := h.DB.CreateReadingList(t.Context(), userID, "Alpha", nil)
	require.NoError(t, err)
	_, err = h.DB.CreateReadingList(t.Context(), userID, "Beta", nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-lists", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingLists(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dtos []readingListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 2)
	require.Equal(t, "Alpha", dtos[0].Name)
}

func TestHandleReadingLists_Create(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	body := mustMarshal(t, readingListRequest{Name: "To Read"})
	r := httptest.NewRequest(http.MethodPost, "/api/reading-lists", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingLists(w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	var dto readingListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "To Read", dto.Name)
}

func TestHandleReadingLists_Create_MissingName(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	body := mustMarshal(t, readingListRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/reading-lists", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingLists(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleReadingLists_Create_Conflict(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	_, err := h.DB.CreateReadingList(t.Context(), userID, "Favorites", nil)
	require.NoError(t, err)

	body := mustMarshal(t, readingListRequest{Name: "Favorites"})
	r := httptest.NewRequest(http.MethodPost, "/api/reading-lists", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingLists(w, r)

	require.Equal(t, http.StatusConflict, w.Code)
}

func TestHandleReadingLists_MethodNotAllowed(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/reading-lists", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingLists(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// --- Get / Update / Delete ---

func TestHandleReadingListRoutes_Get(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "To Read", nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-lists/"+rl.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dto readingListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, rl.ID, dto.ID)
}

func TestHandleReadingListRoutes_Get_NotFound(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-lists/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleReadingListRoutes_Update(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "Old Name", nil)
	require.NoError(t, err)

	body := mustMarshal(t, readingListRequest{Name: "New Name"})
	r := httptest.NewRequest(http.MethodPut, "/api/reading-lists/"+rl.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dto readingListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "New Name", dto.Name)
}

func TestHandleReadingListRoutes_Update_NotFound(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	body := mustMarshal(t, readingListRequest{Name: "Name"})
	r := httptest.NewRequest(http.MethodPut, "/api/reading-lists/nonexistent", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleReadingListRoutes_Update_Conflict(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	_, err := h.DB.CreateReadingList(t.Context(), userID, "First", nil)
	require.NoError(t, err)
	second, err := h.DB.CreateReadingList(t.Context(), userID, "Second", nil)
	require.NoError(t, err)

	body := mustMarshal(t, readingListRequest{Name: "First"})
	r := httptest.NewRequest(http.MethodPut, "/api/reading-lists/"+second.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusConflict, w.Code)
}

func TestHandleReadingListRoutes_Delete(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "To Delete", nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodDelete, "/api/reading-lists/"+rl.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandleReadingListRoutes_Delete_NotFound(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/reading-lists/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

// --- Books sub-resource ---

func TestHandleReadingListRoutes_Books_List(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)
	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err)
	_, err = h.DB.AddBookToReadingList(t.Context(), rl.ID, userID, book.ID)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-lists/"+rl.ID+"/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var result bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Books, 1)
}

func TestHandleReadingListRoutes_Books_List_NotFound(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-lists/nonexistent/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleReadingListRoutes_Books_Add(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)
	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err)

	body := mustMarshal(t, addBookToReadingListRequest{BookID: book.ID})
	r := httptest.NewRequest(http.MethodPost, "/api/reading-lists/"+rl.ID+"/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandleReadingListRoutes_Books_Add_BookNotFound(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)

	body := mustMarshal(t, addBookToReadingListRequest{BookID: "nonexistent-book"})
	r := httptest.NewRequest(http.MethodPost, "/api/reading-lists/"+rl.ID+"/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleReadingListRoutes_Books_Add_ListNotFound(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err)

	body := mustMarshal(t, addBookToReadingListRequest{BookID: book.ID})
	r := httptest.NewRequest(http.MethodPost, "/api/reading-lists/nonexistent/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleReadingListRoutes_Books_Add_MissingBookID(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)

	body := mustMarshal(t, addBookToReadingListRequest{BookID: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/reading-lists/"+rl.ID+"/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleReadingListRoutes_Books_Remove(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)
	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err)
	_, err = h.DB.AddBookToReadingList(t.Context(), rl.ID, userID, book.ID)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodDelete, "/api/reading-lists/"+rl.ID+"/books/"+book.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandleReadingListRoutes_Books_Remove_ListNotFound(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/reading-lists/nonexistent/books/some-book", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleReadingListRoutes_Delete_AuditMetadata(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "My Favorites", nil)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodDelete, "/api/reading-lists/"+rl.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err)

	var deleteLog *db.AuditLog
	for i := range logs {
		if logs[i].Action == db.AuditActionReadingListDeleted {
			deleteLog = &logs[i]
			break
		}
	}
	require.NotNil(t, deleteLog, "expected a reading_list.deleted audit log entry")
	require.NotNil(t, deleteLog.Metadata, "audit log metadata should not be nil")

	var meta map[string]any
	require.NoError(t, json.Unmarshal([]byte(*deleteLog.Metadata), &meta))
	require.Equal(t, "My Favorites", meta["name"])
}

func TestHandleReadingListRoutes_InvalidPath(t *testing.T) {
	h, userID := setupReadingListHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/reading-lists/", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleReadingListRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
