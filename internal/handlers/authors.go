package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
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
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleAuthor handles GET/PUT/DELETE /api/authors/{id}.
func (h *AuthorHandler) HandleAuthor(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/authors/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid author ID")
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
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// listAuthors godoc
// @Summary     List authors
// @Description Returns all authors
// @Tags        Authors
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Success     200 {array}  authorDTO
// @Failure     500 {object} errorResponse
// @Router      /authors [get]
func (h *AuthorHandler) listAuthors(w http.ResponseWriter, r *http.Request) {
	slog.DebugContext(r.Context(), "listing authors")
	authors, err := h.DB.ListAuthors(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list authors", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list authors")
		return
	}

	slog.DebugContext(r.Context(), "authors listed", slog.Int(otelkeys.Count, len(authors)))

	dtos := make([]authorDTO, 0, len(authors))
	for i := range authors {
		dtos = append(dtos, toAuthorDTO(&authors[i]))
	}

	writeJSON(r.Context(), w, http.StatusOK, dtos)
}

// createAuthor godoc
// @Summary     Create an author
// @Description Create a new author
// @Tags        Authors
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       body body     authorRequest true "Author data"
// @Success     201  {object} authorDTO
// @Failure     400  {object} errorResponse
// @Failure     409  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /authors [post]
func (h *AuthorHandler) createAuthor(w http.ResponseWriter, r *http.Request) {
	var req authorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "name is required")
		return
	}

	slog.DebugContext(r.Context(), "creating author", slog.String(otelkeys.Name, req.Name))

	a, err := h.DB.CreateAuthor(r.Context(), req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID, req.ImageURL)
	if err != nil {
		if err == db.ErrAuthorNameExists {
			writeError(r.Context(), w, http.StatusConflict, "an author with that name already exists")
			return
		}
		slog.ErrorContext(r.Context(), "failed to create author", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create author")
		return
	}

	slog.DebugContext(r.Context(), "author created",
		slog.String(otelkeys.AuthorID, a.ID),
		slog.String(otelkeys.Name, a.Name),
	)

	userID := auth.UserIDFromContext(r.Context())
	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionAuthorCreated, "author", a.ID, map[string]any{"name": a.Name}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	writeJSON(r.Context(), w, http.StatusCreated, toAuthorDTO(a))
}

// getAuthor godoc
// @Summary     Get an author
// @Description Returns a single author by ID
// @Tags        Authors
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       id  path     string true "Author ID"
// @Success     200 {object} authorDTO
// @Failure     400 {object} errorResponse
// @Failure     404 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /authors/{id} [get]
func (h *AuthorHandler) getAuthor(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "fetching author", slog.String(otelkeys.AuthorID, id))
	a, err := h.DB.GetAuthor(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "author not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get author", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get author")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, toAuthorDTO(a))
}

// updateAuthor godoc
// @Summary     Update an author
// @Description Update an existing author
// @Tags        Authors
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       id   path     string        true "Author ID"
// @Param       body body     authorRequest true "Author data"
// @Success     200  {object} authorDTO
// @Failure     400  {object} errorResponse
// @Failure     404  {object} errorResponse
// @Failure     409  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /authors/{id} [put]
func (h *AuthorHandler) updateAuthor(w http.ResponseWriter, r *http.Request, id string) {
	var req authorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "name is required")
		return
	}

	slog.DebugContext(r.Context(), "updating author",
		slog.String(otelkeys.AuthorID, id),
		slog.String(otelkeys.Name, req.Name),
	)

	a, err := h.DB.UpdateAuthor(r.Context(), id, req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID, req.ImageURL)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "author not found")
			return
		}
		if err == db.ErrAuthorNameExists {
			writeError(r.Context(), w, http.StatusConflict, "an author with that name already exists")
			return
		}
		slog.ErrorContext(r.Context(), "failed to update author", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to update author")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionAuthorUpdated, "author", a.ID, map[string]any{"name": a.Name}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	writeJSON(r.Context(), w, http.StatusOK, toAuthorDTO(a))
}

// deleteAuthor godoc
// @Summary     Delete an author
// @Description Delete an author by ID
// @Tags        Authors
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       id  path     string true "Author ID"
// @Success     204 "No Content"
// @Failure     400 {object} errorResponse
// @Failure     404 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /authors/{id} [delete]
func (h *AuthorHandler) deleteAuthor(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "deleting author", slog.String(otelkeys.AuthorID, id))

	author, err := h.DB.GetAuthor(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "author not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get author", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete author")
		return
	}

	if err := h.DB.DeleteAuthor(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "author not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete author", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete author")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionAuthorDeleted, "author", id, map[string]any{"name": author.Name}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	w.WriteHeader(http.StatusNoContent)
}
