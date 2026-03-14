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

func (h *BookFileHandler) getBookFile(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "fetching book file", slog.String("book_file_id", id))
	bf, err := h.DB.GetBookFile(id)
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

func (h *BookFileHandler) deleteBookFile(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "deleting book file", slog.String("book_file_id", id))
	err := h.DB.DeleteBookFile(id)
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
