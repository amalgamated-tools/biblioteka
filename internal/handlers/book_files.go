package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/email"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// bookFileMimeTypes maps recognised file extensions to their MIME types.
var bookFileMimeTypes = map[string]string{
	"epub": "application/epub+zip",
	"pdf":  "application/pdf",
	"mobi": "application/x-mobipocket-ebook",
	"azw3": "application/vnd.amazon.mobi8-ebook",
	"azw":  "application/vnd.amazon.mobi8-ebook",
	"cbz":  "application/vnd.comicbook+zip",
	"cbr":  "application/vnd.comicbook-rar",
}

// BookFileHandler holds dependencies for book file endpoints.
type BookFileHandler struct {
	DB      *db.DB
	Emailer email.Sender
}

// HandleBookFile routes GET/DELETE /api/book-files/{id} and
// POST /api/book-files/{id}/send.
func (h *BookFileHandler) HandleBookFile(w http.ResponseWriter, r *http.Request) {
	id, sub, ok := extractPathSegments(r.URL.Path, "/api/book-files/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid book file ID")
		return
	}

	switch sub {
	case "":
		switch r.Method {
		case http.MethodGet:
			h.getBookFile(w, r, id)
		case http.MethodDelete:
			h.deleteBookFile(w, r, id)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case "send":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.sendBookFile(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getBookFile godoc
// @Summary     Get a book file
// @Description Returns a single book file by ID
// @Tags        BookFiles
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       id  path     string true "Book File ID"
// @Success     200 {object} bookFileDTO
// @Failure     400 {object} errorResponse
// @Failure     404 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /book-files/{id} [get]
func (h *BookFileHandler) getBookFile(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "fetching book file", slog.String(otelkeys.BookFileID, id))
	bf, err := h.DB.GetBookFile(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "book file not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get book file", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get book file")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, toBookFileDTO(bf))
}

// deleteBookFile godoc
// @Summary     Delete a book file
// @Description Delete a book file by ID
// @Tags        BookFiles
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       id  path     string true "Book File ID"
// @Success     204 "No Content"
// @Failure     400 {object} errorResponse
// @Failure     404 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /book-files/{id} [delete]
func (h *BookFileHandler) deleteBookFile(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "deleting book file", slog.String(otelkeys.BookFileID, id))

	bf, err := h.DB.GetBookFile(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "book file not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get book file", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete book file")
		return
	}

	if err := h.DB.DeleteBookFile(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "book file not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete book file", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete book file")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionBookFileDeleted, "book_file", id, map[string]any{"book_id": bf.BookID, "file_name": bf.FileName, "file_type": bf.FileType}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	w.WriteHeader(http.StatusNoContent)
}

// sendBookFileRequest is the request body for the send-via-email endpoint.
type sendBookFileRequest struct {
	Email string `json:"email"`
}

// sendBookFile godoc
// @Summary     Send a book file via email
// @Description Sends a book file as an email attachment to the given address.
// @Description Requires SMTP to be configured on the server (SMTP_HOST env var).
// @Tags        BookFiles
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path     string              true "Book File ID"
// @Param       body body     sendBookFileRequest true "Recipient email address"
// @Success     200  {object} map[string]string
// @Failure     400  {object} errorResponse
// @Failure     401  {object} errorResponse
// @Failure     404  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /book-files/{id}/send [post]
func (h *BookFileHandler) sendBookFile(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	slog.DebugContext(ctx, "send book file request", slog.String("book_file_id", id))

	var req sendBookFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "invalid email address")
		return
	}

	bf, err := h.DB.GetBookFile(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "book file not found")
			return
		}
		slog.ErrorContext(ctx, "failed to get book file", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to get book file")
		return
	}

	fileData, err := os.ReadFile(bf.FilePath)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read book file from disk",
			slog.String("book_file_id", id),
			slog.String("path", bf.FilePath),
			slog.Any("error", err),
		)
		writeError(w, http.StatusInternalServerError, "failed to read book file")
		return
	}

	mimeType := bookFileMimeTypes[strings.ToLower(bf.FileType)]
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	subject := fmt.Sprintf("Book: %s", bf.FileName)
	body := fmt.Sprintf("Please find the attached book file: %s", bf.FileName)

	if err := h.Emailer.SendWithAttachment(ctx, req.Email, subject, body, bf.FileName, fileData, mimeType); err != nil {
		slog.ErrorContext(ctx, "failed to send book file email",
			slog.String("book_file_id", id),
			slog.String("to", req.Email),
			slog.Any("error", err),
		)
		writeError(w, http.StatusInternalServerError, "failed to send email")
		return
	}

	slog.InfoContext(ctx, "book file sent via email",
		slog.String("book_file_id", id),
		slog.String("to", req.Email),
		slog.String("file_name", bf.FileName),
	)

	writeJSON(w, http.StatusOK, map[string]string{"message": "email sent"})
}
