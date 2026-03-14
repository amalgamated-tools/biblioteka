package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// BookFileHandler holds dependencies for book file endpoints.
type BookFileHandler struct {
	DB *db.DB
}

// HandleBookFile handles GET/DELETE /api/book-files/{id}.
func (h *BookFileHandler) HandleBookFile(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/book-files/")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid book file ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getBookFile(w, r, id)
	case http.MethodDelete:
		h.deleteBookFile(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
	slog.DebugContext(r.Context(), "fetching book file", slog.String("book_file_id", id))
	bf, err := h.DB.GetBookFile(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "book file not found")
			return
		}
		slog.Error("failed to get book file", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get book file")
		return
	}

	writeJSON(w, http.StatusOK, toBookFileDTO(bf))
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
	slog.DebugContext(r.Context(), "deleting book file", slog.String("book_file_id", id))
	err := h.DB.DeleteBookFile(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "book file not found")
			return
		}
		slog.Error("failed to delete book file", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete book file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
