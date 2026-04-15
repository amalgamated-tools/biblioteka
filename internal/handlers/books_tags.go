package handlers

import "net/http"

// setBookTagsRequest is the request body for setting book tags.
type setBookTagsRequest struct {
	TagIDs []string `json:"tag_ids"`
}

// getBookTags returns the tags associated with the specified book.
func (h *BookHandler) getBookTags(w http.ResponseWriter, r *http.Request, bookID string) {
	respondBookSubResource(r.Context(), w, bookID, h.DB.GetBookTags, toTagDTO, "book tags")
}

// putBookTags replaces the tag list for the specified book.
func (h *BookHandler) putBookTags(w http.ResponseWriter, r *http.Request, bookID string) {
	putBookSubResource(w, r, bookID, h.DB.GetBookTags, h.DB.SetBookTags,
		func(req *setBookTagsRequest) []string { return req.TagIDs },
		toTagDTO, "book tags")
}
