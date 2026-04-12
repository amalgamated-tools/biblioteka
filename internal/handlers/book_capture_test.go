package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/stretchr/testify/require"
)

// setupCaptureHandler creates a BookHandler backed by an in-memory test DB, a
// test library rooted at a temp directory, and a mockEnqueuer. It returns the
// handler, the user ID, and the library ID.
func setupCaptureHandler(t *testing.T) (h *BookHandler, userID, libraryID string) {
	t.Helper()

	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Capture User", "capture@example.com", "password1")
	require.NoError(t, err, "create user")

	libDir := t.TempDir()
	pathsJSON, err := json.Marshal([]string{libDir})
	require.NoError(t, err, "marshal library paths")
	lib, err := d.CreateLibrary(t.Context(), "Capture Library", string(pathsJSON), db.LibraryOrganizationNone, false)
	require.NoError(t, err, "create library")

	h = &BookHandler{
		DB:       d,
		Enqueuer: &mockEnqueuer{},
	}

	return h, user.ID, lib.ID
}

func makeCaptureRequest(t *testing.T, body captureRequest) *http.Request {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodPost, "/api/books/capture", bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestHandleCapture_Success(t *testing.T) {
	h, userID, libraryID := setupCaptureHandler(t)
	enq := &mockEnqueuer{}
	h.Enqueuer = enq

	r := makeCaptureRequest(t, captureRequest{
		URL:       "https://example.com/article",
		LibraryID: libraryID,
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleCapture(w, r)

	require.Equal(t, http.StatusAccepted, w.Code)

	var resp captureAcceptedResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "https://example.com/article", resp.URL)
	require.Equal(t, libraryID, resp.LibraryID)
	require.NotEmpty(t, resp.Message)
}

func TestHandleCapture_EnqueuesCaptureURLJob(t *testing.T) {
	h, userID, libraryID := setupCaptureHandler(t)
	enq := &mockEnqueuer{}
	h.Enqueuer = enq

	r := makeCaptureRequest(t, captureRequest{
		URL:         "https://example.com/article",
		LibraryID:   libraryID,
		Title:       "Override Title",
		Author:      "Override Author",
		Description: "Some description",
		Language:    "en",
		Publisher:   "Test Press",
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleCapture(w, r)

	require.Equal(t, http.StatusAccepted, w.Code)

	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Len(t, enq.jobs, 1, "expected exactly one enqueued job")
	require.Equal(t, jobs.JobCaptureURL, enq.jobs[0].Name)

	var payload jobs.CaptureURLPayload
	require.NoError(t, json.Unmarshal(enq.jobs[0].Payload, &payload))
	require.Equal(t, "https://example.com/article", payload.URL)
	require.Equal(t, libraryID, payload.LibraryID)
	require.Equal(t, userID, payload.UserID)
	require.Equal(t, "Override Title", payload.OverrideTitle)
	require.Equal(t, "Override Author", payload.OverrideAuthor)
	require.Equal(t, "Some description", payload.OverrideDescription)
	require.Equal(t, "en", payload.OverrideLanguage)
	require.Equal(t, "Test Press", payload.OverridePublisher)
	require.NotEmpty(t, payload.LibraryRoot, "library root should be set")
}

func TestHandleCapture_WrongMethod(t *testing.T) {
	h, userID, _ := setupCaptureHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/books/capture", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleCapture(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleCapture_MissingURL(t *testing.T) {
	h, userID, libraryID := setupCaptureHandler(t)

	r := makeCaptureRequest(t, captureRequest{LibraryID: libraryID})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleCapture(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Contains(t, resp.Error, "url")
}

func TestHandleCapture_InvalidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"ftp scheme", "ftp://example.com/file"},
		{"javascript scheme", "javascript:alert(1)"},
		{"no scheme", "example.com/article"},
		{"relative path", "/relative/path"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, userID, libraryID := setupCaptureHandler(t)
			r := makeCaptureRequest(t, captureRequest{URL: tc.url, LibraryID: libraryID})
			r = withUserID(r, userID)
			w := httptest.NewRecorder()
			h.HandleCapture(w, r)
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandleCapture_MissingLibraryID(t *testing.T) {
	h, userID, _ := setupCaptureHandler(t)

	r := makeCaptureRequest(t, captureRequest{URL: "https://example.com/article"})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleCapture(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Contains(t, resp.Error, "library_id")
}

func TestHandleCapture_LibraryNotFound(t *testing.T) {
	h, userID, _ := setupCaptureHandler(t)

	r := makeCaptureRequest(t, captureRequest{
		URL:       "https://example.com/article",
		LibraryID: "nonexistent-library-id",
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleCapture(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleCapture_NoEnqueuer(t *testing.T) {
	h, userID, libraryID := setupCaptureHandler(t)
	h.Enqueuer = nil

	r := makeCaptureRequest(t, captureRequest{
		URL:       "https://example.com/article",
		LibraryID: libraryID,
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleCapture(w, r)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleCapture_EnqueueFailure(t *testing.T) {
	h, userID, libraryID := setupCaptureHandler(t)
	h.Enqueuer = &mockEnqueuer{err: errors.New("redis unavailable")}

	r := makeCaptureRequest(t, captureRequest{
		URL:       "https://example.com/article",
		LibraryID: libraryID,
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleCapture(w, r)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleCapture_AuditLog(t *testing.T) {
	h, userID, libraryID := setupCaptureHandler(t)

	r := makeCaptureRequest(t, captureRequest{
		URL:       "https://example.com/article",
		LibraryID: libraryID,
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleCapture(w, r)

	require.Equal(t, http.StatusAccepted, w.Code)

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err)

	found := false
	for _, l := range logs {
		if l.Action == db.AuditActionBookCaptured {
			found = true
			require.NotNil(t, l.UserID)
			require.Equal(t, userID, *l.UserID)
			break
		}
	}
	require.True(t, found, "expected book.captured audit log entry")
}

func TestIsValidCaptureURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://example.com/article", true},
		{"http://example.com/article", true},
		{"ftp://example.com/file", false},
		{"javascript:alert(1)", false},
		{"example.com/article", false},
		{"/relative/path", false},
		{"", false},
		{"mailto:user@example.com", false},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			require.Equal(t, tc.want, isValidCaptureURL(tc.url))
		})
	}
}
