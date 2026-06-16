package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/filetype"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/smtp"
)

// BookFileHandler holds dependencies for book file endpoints.
type BookFileHandler struct {
	DB *db.DB
	// SendMailFunc overrides the default smtp.Send implementation (used in tests).
	SendMailFunc smtp.SendFunc
	// Secrets decrypts sensitive settings (SMTP password) read from the database.
	// If nil, values are read as-is (backward compatible with plaintext storage).
	Secrets *auth.SecretEncrypter
}

// smtpGetSetting returns a getSetting function that wraps h.DB.GetSetting.
// When h.Secrets is set, the stored SMTP password is decrypted transparently.
func (h *BookFileHandler) smtpGetSetting() func(context.Context, string) (string, error) {
	return makeDecryptingSMTPGetSetting(h.DB.GetSetting, h.Secrets)
}

// HandleBookFile handles GET/DELETE /api/book-files/{id} and sub-resources
// such as /api/book-files/{id}/download and /api/book-files/{id}/email.
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
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case "download":
		if r.Method != http.MethodGet {
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.downloadBookFile(w, r, id)
	case "email":
		h.handleEmailBookFile(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusNotFound, "not found")
	}
}

// readBookFileData reads the contents of the file at filePath from disk.
func readBookFileData(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read book file %q: %w", filePath, err)
	}
	return data, nil
}

// getBookFile returns a single book file record by ID.
//
//	@Summary		Get a book file
//	@Description	Returns a single book file by ID
//	@Tags			BookFiles
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book File ID"
//	@Success		200	{object}	bookFileDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/book-files/{id} [get]
func (h *BookFileHandler) getBookFile(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "fetching book file", slog.String(otelkeys.BookFileID, id))
	bf, err := h.DB.GetBookFile(r.Context(), id)
	if handleDBErr(r.Context(), w, err, "book file") {
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, toBookFileDTO(bf))
}

// deleteBookFile permanently removes a book file record.
//
//	@Summary		Delete a book file
//	@Description	Delete a book file by ID
//	@Tags			BookFiles
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book File ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/book-files/{id} [delete]
func (h *BookFileHandler) deleteBookFile(w http.ResponseWriter, r *http.Request, id string) {
	deleteResource(h.DB, w, r, id, "book file", "book_file", otelkeys.BookFileID,
		h.DB.GetBookFile, h.DB.DeleteBookFile,
		db.AuditActionBookFileDeleted,
		func(bf *db.BookFile) map[string]any {
			return map[string]any{"book_id": bf.BookID, "file_name": bf.FileName, "file_type": bf.FileType}
		},
	)
}

// downloadBookFile serves the actual book file content for download and increments
// the download count.
//
//	@Summary		Download a book file
//	@Description	Serves the book file content for download and increments the download count
//	@Tags			BookFiles
//	@Produce		octet-stream
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book File ID"
//	@Success		200	"File content"
//	@Failure		400	{object}	errorResponse
//	@Failure		403	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/book-files/{id}/download [get]
func (h *BookFileHandler) downloadBookFile(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	bf, err := h.DB.GetBookFile(ctx, id)
	if handleDBErr(ctx, w, err, "book file") {
		return
	}

	allowed, pathErr := isBookFilePathAllowed(ctx, h.DB, bf.FilePath)
	if pathErr != nil {
		slog.ErrorContext(ctx, "failed to validate book file path",
			slog.String(otelkeys.BookFileID, id),
			slog.Any(otelkeys.Error, pathErr),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to validate file path")
		return
	}
	if !allowed {
		slog.WarnContext(ctx, "download blocked: file path outside library roots",
			slog.String(otelkeys.BookFileID, id),
			slog.String(otelkeys.Path, bf.FilePath),
		)
		writeError(ctx, w, http.StatusForbidden, "file path is outside allowed library directories")
		return
	}

	f, err := os.Open(bf.FilePath)
	if err != nil {
		slog.ErrorContext(ctx, "failed to open book file",
			slog.String(otelkeys.BookFileID, id),
			slog.Any(otelkeys.Error, err),
		)
		if errors.Is(err, os.ErrNotExist) {
			writeError(ctx, w, http.StatusNotFound, "book file not found on disk")
		} else {
			writeError(ctx, w, http.StatusInternalServerError, "failed to read file")
		}
		return
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			slog.WarnContext(ctx, "failed to close book file",
				slog.String(otelkeys.BookFileID, id),
				slog.Any(otelkeys.Error, closeErr),
			)
		}
	}()

	stat, err := f.Stat()
	if err != nil {
		slog.ErrorContext(ctx, "failed to stat book file",
			slog.String(otelkeys.BookFileID, id),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to read file")
		return
	}

	// Increment the download count (best-effort; a failure here should not
	// prevent the download from completing).
	if incErr := h.DB.IncrementBookFileDownloadCount(ctx, id); incErr != nil {
		slog.WarnContext(ctx, "failed to increment download count",
			slog.String(otelkeys.BookFileID, id),
			slog.Any(otelkeys.Error, incErr),
		)
	}

	// Record a timestamped download event for the histogram (best-effort).
	// Uses context.WithoutCancel so a client disconnect doesn't prevent
	// recording the event, with a short timeout to bound latency.
	if userID := auth.UserIDFromContext(ctx); userID != "" {
		recCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		if recErr := h.DB.RecordBookDownload(recCtx, id, userID); recErr != nil {
			slog.WarnContext(recCtx, "failed to record book download event",
				slog.String(otelkeys.BookFileID, id),
				slog.String(otelkeys.UserID, userID),
				slog.Any(otelkeys.Error, recErr),
			)
		}
		cancel()
	}

	mimeType := filetype.MIMETypeOrOctetStream(bf.FileType)

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": bf.FileName}))
	http.ServeContent(w, r, bf.FileName, stat.ModTime(), f)
}
