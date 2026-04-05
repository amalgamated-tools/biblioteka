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

func TestGetBookFiles_Empty(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/files", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var files []bookFileDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &files), "unmarshal")
	require.Len(t, files, 0)
}

func TestGetBookFiles_WithFiles(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBookFile(t.Context(), b.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/files", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var files []bookFileDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &files), "unmarshal")
	require.Len(t, files, 1)
	require.Equal(t, "gunslinger.epub", files[0].FileName)
}

func TestPostBookFiles_Success(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, createBookFileRequest{
		FileType: "epub",
		FileName: "gunslinger.epub",
		FileSize: 2048,
		FilePath: "/books/gunslinger.epub",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	var dto bookFileDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "gunslinger.epub", dto.FileName)
	require.Equal(t, "epub", dto.FileType)
	require.Equal(t, int64(2048), dto.FileSize)
	require.Equal(t, b.ID, dto.BookID)
}

func TestPostBookFiles_MissingFileType(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, createBookFileRequest{
		FileName: "gunslinger.epub",
		FilePath: "/books/gunslinger.epub",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostBookFiles_MissingFileName(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, createBookFileRequest{
		FileType: "epub",
		FilePath: "/books/gunslinger.epub",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostBookFiles_MissingFilePath(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, createBookFileRequest{
		FileType: "epub",
		FileName: "gunslinger.epub",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostBookFiles_InvalidJSON(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", strings.NewReader("not-json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBookFiles_MethodNotAllowed(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/files", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestPostBookFiles_AuditLog(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, createBookFileRequest{
		FileType: "epub",
		FileName: "gunslinger.epub",
		FilePath: "/books/gunslinger.epub",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err, "list audit logs")

	found := false
	for _, l := range logs {
		if l.Action == "book_file.created" {
			found = true
			break
		}
	}
	require.True(t, found)
}
