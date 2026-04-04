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

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var files []bookFileDTO
	if err := json.Unmarshal(w.Body.Bytes(), &files); err != nil {
		require.NoError(t, err, "unmarshal")
	}
	if len(files) != 0 {
		t.Errorf("len = %d, want 0", len(files))
	}
}

func TestGetBookFiles_WithFiles(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	if _, err := h.DB.CreateBookFile(t.Context(), b.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub"); err != nil {
		require.NoError(t, err, "create book file")
	}

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/files", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var files []bookFileDTO
	if err := json.Unmarshal(w.Body.Bytes(), &files); err != nil {
		require.NoError(t, err, "unmarshal")
	}
	if len(files) != 1 {
		require.Failf(t, "failed", "len = %d, want 1", len(files))
	}
	if files[0].FileName != "gunslinger.epub" {
		t.Errorf("file_name = %q, want %q", files[0].FileName, "gunslinger.epub")
	}
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

	if w.Code != http.StatusCreated {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var dto bookFileDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		require.NoError(t, err, "unmarshal")
	}
	if dto.FileName != "gunslinger.epub" {
		t.Errorf("file_name = %q, want %q", dto.FileName, "gunslinger.epub")
	}
	if dto.FileType != "epub" {
		t.Errorf("file_type = %q, want %q", dto.FileType, "epub")
	}
	if dto.FileSize != 2048 {
		t.Errorf("file_size = %d, want 2048", dto.FileSize)
	}
	if dto.BookID != b.ID {
		t.Errorf("book_id = %q, want %q", dto.BookID, b.ID)
	}
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

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
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

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
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

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestPostBookFiles_InvalidJSON(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", strings.NewReader("not-json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBookFiles_MethodNotAllowed(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/files", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
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

	if w.Code != http.StatusCreated {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err, "list audit logs")

	found := false
	for _, l := range logs {
		if l.Action == "book_file.created" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit log entry with action book_file.created")
	}
}
