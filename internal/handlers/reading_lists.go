package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ReadingListHandler holds dependencies for reading list endpoints.
type ReadingListHandler struct {
	DB *db.DB
}

type readingListRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type addBookToReadingListRequest struct {
	BookID string `json:"book_id"`
}

type readingListDTO struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description *string      `json:"description"`
	BookCount   int          `json:"book_count"`
	CreatedAt   db.Timestamp `json:"created_at"`
	UpdatedAt   db.Timestamp `json:"updated_at"`
}

func toReadingListDTO(rl *db.ReadingList) readingListDTO {
	return readingListDTO{
		ID:          rl.ID,
		Name:        rl.Name,
		Description: rl.Description,
		BookCount:   rl.BookCount,
		CreatedAt:   rl.CreatedAt,
		UpdatedAt:   rl.UpdatedAt,
	}
}

// readingListOps returns the userOwnedNamedEntityOps configuration for the ReadingList entity.
func (h *ReadingListHandler) readingListOps() userOwnedNamedEntityOps[db.ReadingList, readingListDTO, readingListRequest] {
	return userOwnedNamedEntityOps[db.ReadingList, readingListDTO, readingListRequest]{
		db:              h.DB,
		entityLabel:     "reading list",
		auditEntityType: "reading_list",
		entityArticle:   "a reading list",
		idKey:           otelkeys.ReadingListID,
		errInvalidName:  db.ErrInvalidReadingListName,
		errNameExists:   db.ErrReadingListNameExists,
		auditCreate:     db.AuditActionReadingListCreated,
		auditUpdate:     db.AuditActionReadingListUpdated,
		get:             h.DB.GetReadingList,
		create: func(ctx context.Context, userID string, req readingListRequest) (*db.ReadingList, error) {
			return h.DB.CreateReadingList(ctx, userID, req.Name, req.Description)
		},
		update: func(ctx context.Context, id, userID string, req readingListRequest) (*db.ReadingList, error) {
			return h.DB.UpdateReadingList(ctx, id, userID, req.Name, req.Description)
		},
		reqName:    func(req readingListRequest) string { return req.Name },
		entityName: func(rl *db.ReadingList) string { return rl.Name },
		entityID:   func(rl *db.ReadingList) string { return rl.ID },
		toDTO:      toReadingListDTO,
	}
}

// HandleReadingLists handles GET /api/reading-lists and POST /api/reading-lists.
func (h *ReadingListHandler) HandleReadingLists(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listReadingLists(w, r)
	case http.MethodPost:
		h.createReadingList(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleReadingListRoutes dispatches /api/reading-lists/{id},
// /api/reading-lists/{id}/books, and /api/reading-lists/{id}/books/{bookId}.
func (h *ReadingListHandler) HandleReadingListRoutes(w http.ResponseWriter, r *http.Request) {
	id, sub, ok := extractPathSegments(r.URL.Path, "/api/reading-lists/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid reading list ID")
		return
	}

	switch {
	case sub == "":
		h.handleReadingList(w, r, id)
	case sub == "books":
		h.handleReadingListBooks(w, r, id)
	case strings.HasPrefix(sub, "books/"):
		bookID := strings.TrimPrefix(sub, "books/")
		if bookID == "" {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid book ID")
			return
		}
		h.handleReadingListBook(w, r, id, bookID)
	default:
		writeError(r.Context(), w, http.StatusNotFound, "not found")
	}
}

// listReadingLists returns all reading lists for the authenticated user.
//
//	@Summary		List reading lists
//	@Description	Returns all reading lists owned by the authenticated user
//	@Tags			Reading Lists
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Success		200	{array}		readingListDTO
//	@Failure		500	{object}	errorResponse
//	@Router			/reading-lists [get]
func (h *ReadingListHandler) listReadingLists(w http.ResponseWriter, r *http.Request) {
	listUserEntities(w, r, "reading lists", h.DB.ListReadingLists, toReadingListDTO)
}

// createReadingList creates a new reading list for the authenticated user.
//
//	@Summary		Create a reading list
//	@Description	Create a new reading list for the authenticated user
//	@Tags			Reading Lists
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			body	body		readingListRequest	true	"Reading list data"
//	@Success		201		{object}	readingListDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/reading-lists [post]
func (h *ReadingListHandler) createReadingList(w http.ResponseWriter, r *http.Request) {
	createUserOwnedNamedEntity(h.readingListOps(), w, r)
}

// handleReadingList dispatches GET, PUT, DELETE for /api/reading-lists/{id}.
func (h *ReadingListHandler) handleReadingList(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		h.getReadingList(w, r, id)
	case http.MethodPut:
		h.updateReadingList(w, r, id)
	case http.MethodDelete:
		h.deleteReadingList(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getReadingList returns a single reading list by ID scoped to the authenticated user.
//
//	@Summary		Get a reading list
//	@Description	Returns a single reading list by ID for the authenticated user
//	@Tags			Reading Lists
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Reading list ID"
//	@Success		200	{object}	readingListDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/reading-lists/{id} [get]
func (h *ReadingListHandler) getReadingList(w http.ResponseWriter, r *http.Request, id string) {
	getUserOwnedNamedEntity(h.readingListOps(), w, r, id)
}

// updateReadingList updates the name and description of a reading list.
//
//	@Summary		Update a reading list
//	@Description	Update the name and description of a reading list owned by the authenticated user
//	@Tags			Reading Lists
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string				true	"Reading list ID"
//	@Param			body	body		readingListRequest	true	"Reading list data"
//	@Success		200		{object}	readingListDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/reading-lists/{id} [put]
func (h *ReadingListHandler) updateReadingList(w http.ResponseWriter, r *http.Request, id string) {
	updateUserOwnedNamedEntity(h.readingListOps(), w, r, id)
}

// deleteReadingList deletes a reading list owned by the authenticated user.
//
//	@Summary		Delete a reading list
//	@Description	Delete a reading list owned by the authenticated user
//	@Tags			Reading Lists
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Reading list ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/reading-lists/{id} [delete]
func (h *ReadingListHandler) deleteReadingList(w http.ResponseWriter, r *http.Request, id string) {
	deleteUserOwnedResource(h.DB, w, r, id, "reading list", "reading_list", otelkeys.ReadingListID,
		h.DB.GetReadingList, h.DB.DeleteReadingList,
		db.AuditActionReadingListDeleted,
		func(rl *db.ReadingList) map[string]any { return map[string]any{"name": rl.Name} },
	)
}

// handleReadingListBooks dispatches GET and POST for /api/reading-lists/{id}/books.
func (h *ReadingListHandler) handleReadingListBooks(w http.ResponseWriter, r *http.Request, listID string) {
	switch r.Method {
	case http.MethodGet:
		h.listReadingListBooks(w, r, listID)
	case http.MethodPost:
		h.addBookToReadingList(w, r, listID)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// listReadingListBooks returns a paginated list of books in a reading list.
//
//	@Summary		List books in a reading list
//	@Description	Returns a paginated list of books in a reading list owned by the authenticated user
//	@Tags			Reading Lists
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string	true	"Reading list ID"
//	@Param			limit	query		int		false	"Max items per page (default 50, max 200)"
//	@Param			offset	query		int		false	"Number of items to skip (default 0)"
//	@Success		200		{object}	bookListDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/reading-lists/{id}/books [get]
func (h *ReadingListHandler) listReadingListBooks(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	limit, offset := parseLimitOffset(r, defaultPageLimit, maxPageLimit)

	books, total, err := h.DB.ListReadingListBooks(ctx, listID, userID, limit, offset)
	if handleOpErr(ctx, w, err, "reading list", "failed to list reading list books",
		slog.String(otelkeys.ReadingListID, listID),
	) {
		return
	}

	writeJSON(ctx, w, http.StatusOK, bookListDTO{
		Books: mapSlice(books, toBookSummaryDTO),
		paginationMeta: paginationMeta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// addBookToReadingList adds a book to a reading list (idempotent).
//
//	@Summary		Add a book to a reading list
//	@Description	Add a book to a reading list owned by the authenticated user (idempotent)
//	@Tags			Reading Lists
//	@Accept			json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string						true	"Reading list ID"
//	@Param			body	body		addBookToReadingListRequest	true	"Book to add"
//	@Success		204		"No Content"
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/reading-lists/{id}/books [post]
func (h *ReadingListHandler) addBookToReadingList(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()
	var req addBookToReadingListRequest
	if !decodeJSON(r, w, &req) {
		return
	}
	if req.BookID == "" {
		writeError(ctx, w, http.StatusBadRequest, "book_id is required")
		return
	}

	userID := auth.UserIDFromContext(ctx)
	added, err := h.DB.AddBookToReadingList(ctx, listID, userID, req.BookID)
	if errors.Is(err, db.ErrBookNotFound) {
		writeError(ctx, w, http.StatusNotFound, "book not found")
		return
	}
	if handleOpErr(ctx, w, err, "reading list", "failed to add book to reading list",
		slog.String(otelkeys.ReadingListID, listID),
		slog.String(otelkeys.BookID, req.BookID),
	) {
		return
	}

	if added {
		logAudit(ctx, h.DB, userID, db.AuditActionReadingListBookAdded, "reading_list", listID,
			map[string]any{"book_id": req.BookID},
		)
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleReadingListBook dispatches DELETE for /api/reading-lists/{id}/books/{bookId}.
func (h *ReadingListHandler) handleReadingListBook(w http.ResponseWriter, r *http.Request, listID, bookID string) {
	switch r.Method {
	case http.MethodDelete:
		h.removeBookFromReadingList(w, r, listID, bookID)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// removeBookFromReadingList removes a book from a reading list.
//
//	@Summary		Remove a book from a reading list
//	@Description	Remove a book from a reading list owned by the authenticated user
//	@Tags			Reading Lists
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string	true	"Reading list ID"
//	@Param			bookID	path		string	true	"Book ID"
//	@Success		204		"No Content"
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/reading-lists/{id}/books/{bookID} [delete]
func (h *ReadingListHandler) removeBookFromReadingList(w http.ResponseWriter, r *http.Request, listID, bookID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	removed, err := h.DB.RemoveBookFromReadingList(ctx, listID, userID, bookID)
	if handleOpErr(ctx, w, err, "reading list", "failed to remove book from reading list",
		slog.String(otelkeys.ReadingListID, listID),
		slog.String(otelkeys.BookID, bookID),
	) {
		return
	}

	if removed {
		logAudit(ctx, h.DB, userID, db.AuditActionReadingListBookRemoved, "reading_list", listID,
			map[string]any{"book_id": bookID},
		)
	}

	w.WriteHeader(http.StatusNoContent)
}
