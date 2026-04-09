package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	netsmtp "net/smtp"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/smtp"
	"github.com/stretchr/testify/require"
)

// setupEmailHandler creates a BookFileHandler with SMTP configured in DB,
// a book, and a real file on disk.
func setupEmailHandler(t *testing.T) (*BookFileHandler, string, *db.BookFile, string) {
	t.Helper()
	h, userID := setupBookFileHandler(t)

	// Configure SMTP in DB.
	require.NoError(t, h.DB.SetSetting(t.Context(), smtp.SettingKeyHost, "smtp.example.com"))
	require.NoError(t, h.DB.SetSetting(t.Context(), smtp.SettingKeyPort, "587"))
	require.NoError(t, h.DB.SetSetting(t.Context(), smtp.SettingKeyFrom, "noreply@example.com"))
	require.NoError(t, h.DB.SetSetting(t.Context(), smtp.SettingKeyTLS, "starttls"))

	// Write a real file to a temp directory.
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.epub")
	require.NoError(t, os.WriteFile(filePath, []byte("fake epub bytes"), 0o600))

	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Test Book"})
	require.NoError(t, err, "create book")
	bf, err := h.DB.CreateBookFile(t.Context(), book.ID, "epub", "test.epub", 15, nil, filePath)
	require.NoError(t, err, "create book file")

	return h, userID, bf, filePath
}

func TestHandleEmailBookFile_Success(t *testing.T) {
	h, userID, bf, _ := setupEmailHandler(t)

	var calledFrom, calledTo string
	var calledMsg []byte
	h.SendMailFunc = func(_ context.Context, addr string, _ netsmtp.Auth, from, to string, msg []byte, _ string) error {
		calledFrom = from
		calledTo = to
		calledMsg = msg
		return nil
	}

	body := `{"to":"reader@example.com"}`
	r := httptest.NewRequest(http.MethodPost, "/api/book-files/"+bf.ID+"/email", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	require.Equal(t, "noreply@example.com", calledFrom)
	require.Equal(t, "reader@example.com", calledTo)
	require.Contains(t, string(calledMsg), "test.epub")
}

func TestHandleEmailBookFile_MethodNotAllowed(t *testing.T) {
	h, userID, bf, _ := setupEmailHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/"+bf.ID+"/email", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleEmailBookFile_InvalidJSON(t *testing.T) {
	h, userID, bf, _ := setupEmailHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/book-files/"+bf.ID+"/email", bytes.NewBufferString("not json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.SendMailFunc = func(_ context.Context, _ string, _ netsmtp.Auth, _, _ string, _ []byte, _ string) error {
		return nil
	}

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleEmailBookFile_MissingTo(t *testing.T) {
	h, userID, bf, _ := setupEmailHandler(t)
	h.SendMailFunc = func(_ context.Context, _ string, _ netsmtp.Auth, _, _ string, _ []byte, _ string) error {
		return nil
	}

	r := httptest.NewRequest(http.MethodPost, "/api/book-files/"+bf.ID+"/email", bytes.NewBufferString(`{"to":""}`))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "to address is required", resp.Error)
}

func TestHandleEmailBookFile_InvalidToAddress(t *testing.T) {
	h, userID, bf, _ := setupEmailHandler(t)
	h.SendMailFunc = func(_ context.Context, _ string, _ netsmtp.Auth, _, _ string, _ []byte, _ string) error {
		return nil
	}

	r := httptest.NewRequest(http.MethodPost, "/api/book-files/"+bf.ID+"/email", bytes.NewBufferString(`{"to":"not-an-email"}`))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "invalid email address", resp.Error)
}

func TestHandleEmailBookFile_BookFileNotFound(t *testing.T) {
	h, userID, _, _ := setupEmailHandler(t)
	h.SendMailFunc = func(_ context.Context, _ string, _ netsmtp.Auth, _, _ string, _ []byte, _ string) error {
		return nil
	}

	r := httptest.NewRequest(http.MethodPost, "/api/book-files/nonexistent/email", bytes.NewBufferString(`{"to":"reader@example.com"}`))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleEmailBookFile_SMTPNotConfigured(t *testing.T) {
	h, userID := setupBookFileHandler(t)
	// No SMTP settings.

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.epub")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0o600))
	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Test Book"})
	require.NoError(t, err)
	bf, err := h.DB.CreateBookFile(t.Context(), book.ID, "epub", "test.epub", 7, nil, filePath)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/api/book-files/"+bf.ID+"/email", bytes.NewBufferString(`{"to":"reader@example.com"}`))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "SMTP is not configured", resp.Error)
}

func TestHandleEmailBookFile_SendFailure(t *testing.T) {
	h, userID, bf, _ := setupEmailHandler(t)
	h.SendMailFunc = func(_ context.Context, _ string, _ netsmtp.Auth, _, _ string, _ []byte, _ string) error {
		return fmt.Errorf("connection refused")
	}

	r := httptest.NewRequest(http.MethodPost, "/api/book-files/"+bf.ID+"/email", bytes.NewBufferString(`{"to":"reader@example.com"}`))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusBadGateway, w.Code)
}

func TestHandleEmailBookFile_UnknownSubResource(t *testing.T) {
	h, userID, bf, _ := setupEmailHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/book-files/"+bf.ID+"/unknown", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleEmailBookFile_InvalidControlCharsInTo(t *testing.T) {
	h, userID, bf, _ := setupEmailHandler(t)
	h.SendMailFunc = func(_ context.Context, _ string, _ netsmtp.Auth, _, _ string, _ []byte, _ string) error {
		return nil
	}

	body := "{\"to\":\"reader@example.com\r\n\"}"
	r := httptest.NewRequest(http.MethodPost, "/api/book-files/"+bf.ID+"/email", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
