package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// AuthorHandler holds dependencies for author endpoints.
type AuthorHandler struct {
	DB *db.DB
}

type authorRequest struct {
	Name          string  `json:"name"`
	GoodreadsID   *string `json:"goodreads_id"`
	HardcoverID   *string `json:"hardcover_id"`
	GoogleBooksID *string `json:"google_books_id"`
	ImageURL      *string `json:"image_url"`
}

type authorDTO struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	GoodreadsID   *string      `json:"goodreads_id"`
	HardcoverID   *string      `json:"hardcover_id"`
	GoogleBooksID *string      `json:"google_books_id"`
	ImageURL      *string      `json:"image_url"`
	CreatedAt     db.Timestamp `json:"created_at"`
	UpdatedAt     db.Timestamp `json:"updated_at"`
}

func toAuthorDTO(a *db.Author) authorDTO {
	return authorDTO{
		ID:            a.ID,
		Name:          a.Name,
		GoodreadsID:   a.GoodreadsID,
		HardcoverID:   a.HardcoverID,
		GoogleBooksID: a.GoogleBooksID,
		ImageURL:      a.ImageURL,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

// HandleAuthors handles GET /api/authors and POST /api/authors.
func (h *AuthorHandler) HandleAuthors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listAuthors(w, r)
	case http.MethodPost:
		h.createAuthor(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleAuthor handles GET/PUT/DELETE /api/authors/{id}.
func (h *AuthorHandler) HandleAuthor(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/authors/")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid author ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getAuthor(w, r, id)
	case http.MethodPut:
		h.updateAuthor(w, r, id)
	case http.MethodDelete:
		h.deleteAuthor(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *AuthorHandler) listAuthors(w http.ResponseWriter, _ *http.Request) {
	slog.Debug("listing authors")
	authors, err := h.DB.ListAuthors()
	if err != nil {
		slog.Error("failed to list authors", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list authors")
		return
	}

	slog.Debug("authors listed", slog.Int("count", len(authors)))

	dtos := make([]authorDTO, 0, len(authors))
	for i := range authors {
		dtos = append(dtos, toAuthorDTO(&authors[i]))
	}

	writeJSON(w, http.StatusOK, dtos)
}

func (h *AuthorHandler) createAuthor(w http.ResponseWriter, r *http.Request) {
	var req authorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	slog.DebugContext(r.Context(), "creating author", slog.String("name", req.Name))

	a, err := h.DB.CreateAuthor(req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID, req.ImageURL)
	if err != nil {
		if err == db.ErrAuthorNameExists {
			writeError(w, http.StatusConflict, "an author with that name already exists")
			return
		}
		slog.Error("failed to create author", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create author")
		return
	}

	slog.DebugContext(r.Context(), "author created", slog.String("author_id", a.ID), slog.String("name", a.Name))
	writeJSON(w, http.StatusCreated, toAuthorDTO(a))
}

func (h *AuthorHandler) getAuthor(w http.ResponseWriter, _ *http.Request, id string) {
	slog.Debug("fetching author", slog.String("author_id", id))
	a, err := h.DB.GetAuthor(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "author not found")
			return
		}
		slog.Error("failed to get author", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get author")
		return
	}

	writeJSON(w, http.StatusOK, toAuthorDTO(a))
}

func (h *AuthorHandler) updateAuthor(w http.ResponseWriter, r *http.Request, id string) {
	var req authorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	slog.DebugContext(r.Context(), "updating author", slog.String("author_id", id), slog.String("name", req.Name))

	a, err := h.DB.UpdateAuthor(id, req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID, req.ImageURL)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "author not found")
			return
		}
		if err == db.ErrAuthorNameExists {
			writeError(w, http.StatusConflict, "an author with that name already exists")
			return
		}
		slog.Error("failed to update author", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update author")
		return
	}

	writeJSON(w, http.StatusOK, toAuthorDTO(a))
}

func (h *AuthorHandler) deleteAuthor(w http.ResponseWriter, _ *http.Request, id string) {
	slog.Debug("deleting author", slog.String("author_id", id))
	err := h.DB.DeleteAuthor(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "author not found")
			return
		}
		slog.Error("failed to delete author", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete author")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
