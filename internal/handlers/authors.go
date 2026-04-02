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
		db:              h.DB,
		entityLabel:     "author",
		entityArticle:   "an author",
		idKey:           otelkeys.AuthorID,
		errInvalidName:  db.ErrInvalidAuthorName,
		errNameExists:   db.ErrAuthorNameExists,
		auditCreate:     db.AuditActionAuthorCreated,
		auditUpdate:     db.AuditActionAuthorUpdated,
		auditDelete:     db.AuditActionAuthorDeleted,
		pathPrefix:      "/api/authors/",
		collectionLabel: "authors",
		get:             h.DB.GetAuthor,
		list:            h.DB.ListAuthors,
		del:             h.DB.DeleteAuthor,
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
func (h *AuthorHandler) HandleAuthors(w http.ResponseWriter, r *http.Request) {
	handleNamedEntityCollection(h.authorOps(), w, r)
}

// HandleAuthor handles GET/PUT/DELETE /api/authors/{id}.
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
func (h *AuthorHandler) HandleAuthor(w http.ResponseWriter, r *http.Request) {
	handleNamedEntitySingle(h.authorOps(), w, r)
}
