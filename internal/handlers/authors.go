package handlers

import (
	"context"
	"net/http"

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

// authorOps returns the namedEntityOps configuration for the Author entity.
func (h *AuthorHandler) authorOps() namedEntityOps[db.Author, authorDTO, authorRequest] {
	return namedEntityOps[db.Author, authorDTO, authorRequest]{
		db:             h.DB,
		entityLabel:    "author",
		entityArticle:  "an author",
		idKey:          otelkeys.AuthorID,
		errInvalidName: db.ErrInvalidAuthorName,
		errNameExists:  db.ErrAuthorNameExists,
		auditCreate:    db.AuditActionAuthorCreated,
		auditUpdate:    db.AuditActionAuthorUpdated,
		get:            h.DB.GetAuthor,
		create: func(ctx context.Context, req authorRequest) (*db.Author, error) {
			return h.DB.CreateAuthor(ctx, req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID, req.ImageURL)
		},
		update: func(ctx context.Context, id string, req authorRequest) (*db.Author, error) {
			return h.DB.UpdateAuthor(ctx, id, req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID, req.ImageURL)
		},
		reqName:    func(req authorRequest) string { return req.Name },
		entityName: func(a *db.Author) string { return a.Name },
		entityID:   func(a *db.Author) string { return a.ID },
		toDTO:      toAuthorDTO,
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
//
//	@Summary		List authors
//	@Description	Returns all authors
//	@Tags			Authors
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Success		200	{array}		authorDTO
//	@Failure		500	{object}	errorResponse
//	@Router			/authors [get]
func (h *AuthorHandler) listAuthors(w http.ResponseWriter, r *http.Request) {
	listEntities(w, r, "authors", h.DB.ListAuthors, toAuthorDTO)
}

// createAuthor godoc
//
//	@Summary		Create an author
//	@Description	Create a new author
//	@Tags			Authors
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			body	body		authorRequest	true	"Author data"
//	@Success		201		{object}	authorDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/authors [post]
func (h *AuthorHandler) createAuthor(w http.ResponseWriter, r *http.Request) {
	createNamedEntity(h.authorOps(), w, r)
}

// getAuthor godoc
//
//	@Summary		Get an author
//	@Description	Returns a single author by ID
//	@Tags			Authors
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Author ID"
//	@Success		200	{object}	authorDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/authors/{id} [get]
func (h *AuthorHandler) getAuthor(w http.ResponseWriter, r *http.Request, id string) {
	getNamedEntity(h.authorOps(), w, r, id)
}

// updateAuthor godoc
//
//	@Summary		Update an author
//	@Description	Update an existing author
//	@Tags			Authors
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string			true	"Author ID"
//	@Param			body	body		authorRequest	true	"Author data"
//	@Success		200		{object}	authorDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/authors/{id} [put]
func (h *AuthorHandler) updateAuthor(w http.ResponseWriter, r *http.Request, id string) {
	updateNamedEntity(h.authorOps(), w, r, id)
}

// deleteAuthor godoc
//
//	@Summary		Delete an author
//	@Description	Delete an author by ID
//	@Tags			Authors
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Author ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/authors/{id} [delete]
func (h *AuthorHandler) deleteAuthor(w http.ResponseWriter, r *http.Request, id string) {
	deleteResource(h.DB, w, r, id, "author", otelkeys.AuthorID,
		h.DB.GetAuthor, h.DB.DeleteAuthor,
		db.AuditActionAuthorDeleted,
		func(a *db.Author) map[string]any { return map[string]any{"name": a.Name} },
	)
}
