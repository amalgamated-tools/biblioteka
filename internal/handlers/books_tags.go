package handlers

import (
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// setBookTagsRequest is the request body for setting book tags.
type setBookTagsRequest struct {
	TagIDs []string `json:"tag_ids"`
}

// getBookTags returns the tags associated with the specified book.
//
//	@Summary		List book tags
//	@Description	Get the list of tags for a book
//	@Tags			Books
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{array}		tagDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/tags [get]
func (h *BookHandler) getBookTags(w http.ResponseWriter, r *http.Request, bookID string) {
	respondBookSubResource(r.Context(), w, bookID, h.DB.GetBookTags, toTagDTO, "book tags")
}

// putBookTags replaces the tag list for the specified book.
//
//	@Summary		Set book tags
//	@Description	Replace the list of tags for a book
//	@Tags			Books
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string				true	"Book ID"
//	@Param			body	body		setBookTagsRequest	true	"Tag IDs"
//	@Success		200		{array}		tagDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/books/{id}/tags [put]
func (h *BookHandler) putBookTags(w http.ResponseWriter, r *http.Request, bookID string) {
	putBookSubResource(w, r, bookID, h.DB.GetBookTags, h.DB.SetBookTags,
		func(req *setBookTagsRequest) []string { return req.TagIDs },
		toTagDTO, "book tags", h.DB, db.AuditActionBookTagsUpdated)
}
