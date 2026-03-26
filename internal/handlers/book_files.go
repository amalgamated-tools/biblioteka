package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// BookFileHandler holds dependencies for book file endpoints.
type BookFileHandler struct {
	DB *db.DB
}

// HandleBookFile handles GET/DELETE /api/book-files/{id}.
func (h *BookFileHandler) HandleBookFile(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/book-files/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid book file ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getBookFile(w, r, id)
	case http.MethodDelete:
		h.deleteBookFile(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getBookFile godoc
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

// deleteBookFile godoc
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
	slog.DebugContext(r.Context(), "deleting book file", slog.String(otelkeys.BookFileID, id))

	bf, err := h.DB.GetBookFile(r.Context(), id)
	if handleDBErr(r.Context(), w, err, "book file") {
		return
	}

	if err := h.DB.DeleteBookFile(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusNotFound, "book file not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete book file", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete book file")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	logAudit(r.Context(), h.DB, userID, db.AuditActionBookFileDeleted, "book_file", id, map[string]any{"book_id": bf.BookID, "file_name": bf.FileName, "file_type": bf.FileType})

	w.WriteHeader(http.StatusNoContent)
}
