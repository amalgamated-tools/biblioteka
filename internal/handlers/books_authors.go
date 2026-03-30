package handlers

import "net/http"

// setBookAuthorsRequest is the request body for setting book authors.
type setBookAuthorsRequest struct {
	AuthorIDs []string `json:"author_ids"`
}

// getBookAuthors godoc
//
//	@Summary		List book authors
//	@Description	Get the list of authors for a book
//	@Tags			Books
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{array}		authorDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/authors [get]
func (h *BookHandler) getBookAuthors(w http.ResponseWriter, r *http.Request, bookID string) {
	respondBookSubResource(r.Context(), w, bookID, h.DB.GetBookAuthors, toAuthorDTO, "book authors")
}

// putBookAuthors godoc
//
//	@Summary		Set book authors
//	@Description	Replace the list of authors for a book
//	@Tags			Books
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string					true	"Book ID"
//	@Param			body	body		setBookAuthorsRequest	true	"Author IDs"
//	@Success		200		{array}		authorDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/books/{id}/authors [put]
func (h *BookHandler) putBookAuthors(w http.ResponseWriter, r *http.Request, bookID string) {
	putBookSubResource(w, r, bookID, h.DB.GetBookAuthors, h.DB.SetBookAuthors,
		func(req *setBookAuthorsRequest) []string { return req.AuthorIDs },
		toAuthorDTO, "book authors")
}
