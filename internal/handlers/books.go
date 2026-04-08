package handlers

import (
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
)

// BookHandler holds dependencies for book endpoints.
type BookHandler struct {
	DB       *db.DB
	Enqueuer jobs.Enqueuer
}

// HandleBooks handles GET /api/books and POST /api/books.
func (h *BookHandler) HandleBooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listBooks(w, r)
	case http.MethodPost:
		h.createBook(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleBookRoutes dispatches /api/books/{id} and /api/books/{id}/{sub-resource}.
func (h *BookHandler) HandleBookRoutes(w http.ResponseWriter, r *http.Request) {
	id, sub, ok := extractPathSegments(r.URL.Path, "/api/books/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid book ID")
		return
	}

	switch sub {
	case "":
		h.handleBook(w, r, id)
	case "authors":
		switch r.Method {
		case http.MethodGet:
			h.getBookAuthors(w, r, id)
		case http.MethodPut:
			h.putBookAuthors(w, r, id)
		default:
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case "series":
		switch r.Method {
		case http.MethodGet:
			h.getBookSeries(w, r, id)
		case http.MethodPut:
			h.putBookSeries(w, r, id)
		default:
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case "files":
		switch r.Method {
		case http.MethodGet:
			h.getBookFiles(w, r, id)
		case http.MethodPost:
			h.postBookFiles(w, r, id)
		default:
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		}
	default:
		writeError(r.Context(), w, http.StatusNotFound, "not found")
	}
}

func (h *BookHandler) handleBook(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		h.getBook(w, r, id)
	case http.MethodPut:
		h.updateBook(w, r, id)
	case http.MethodDelete:
		h.deleteBook(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
