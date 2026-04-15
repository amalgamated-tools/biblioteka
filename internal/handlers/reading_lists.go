package handlers

import (
	"context"
	"database/sql"
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

func handleReadingListOpErr(ctx context.Context, w http.ResponseWriter, err error, op string, attrs ...slog.Attr) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeError(ctx, w, http.StatusNotFound, "reading list not found")
		return true
	}

	logAttrs := make([]any, 0, len(attrs)+1)
	for _, attr := range attrs {
		logAttrs = append(logAttrs, attr)
	}
	logAttrs = append(logAttrs, slog.Any(otelkeys.Error, err))

	slog.ErrorContext(ctx, op, logAttrs...)
	writeError(ctx, w, http.StatusInternalServerError, op)
	return true
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
func (h *ReadingListHandler) listReadingLists(w http.ResponseWriter, r *http.Request) {
	listUserEntities(w, r, "reading lists", h.DB.ListReadingLists, toReadingListDTO)
}

// createReadingList creates a new reading list for the authenticated user.
func (h *ReadingListHandler) createReadingList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req readingListRequest
	if !decodeJSON(r, w, &req) {
		return
	}
	if !validateName(ctx, w, req.Name) {
		return
	}

	userID := auth.UserIDFromContext(ctx)
	slog.DebugContext(ctx, "creating reading list",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.ReadingListName, req.Name),
	)

	rl, err := h.DB.CreateReadingList(ctx, userID, req.Name, req.Description)
	if err != nil {
		if handleNameErr(ctx, w, err, db.ErrInvalidReadingListName, db.ErrReadingListNameExists, "a reading list") {
			return
		}
		slog.ErrorContext(ctx, "failed to create reading list", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to create reading list")
		return
	}

	logAudit(ctx, h.DB, userID, db.AuditActionReadingListCreated, "reading_list", rl.ID,
		map[string]any{"name": rl.Name},
	)

	writeJSON(ctx, w, http.StatusCreated, toReadingListDTO(rl))
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
func (h *ReadingListHandler) getReadingList(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	rl, err := h.DB.GetReadingList(ctx, id, userID)
	if handleDBErr(ctx, w, err, "reading list") {
		return
	}
	writeJSON(ctx, w, http.StatusOK, toReadingListDTO(rl))
}

// updateReadingList updates the name and description of a reading list.
func (h *ReadingListHandler) updateReadingList(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	var req readingListRequest
	if !decodeJSON(r, w, &req) {
		return
	}
	if !validateName(ctx, w, req.Name) {
		return
	}

	userID := auth.UserIDFromContext(ctx)
	rl, err := h.DB.UpdateReadingList(ctx, id, userID, req.Name, req.Description)
	if handleUpdateErr(ctx, w, err, db.ErrInvalidReadingListName, db.ErrReadingListNameExists, "a reading list", "reading list", id) {
		return
	}

	logAudit(ctx, h.DB, userID, db.AuditActionReadingListUpdated, "reading_list", rl.ID,
		map[string]any{"name": rl.Name},
	)

	writeJSON(ctx, w, http.StatusOK, toReadingListDTO(rl))
}

// deleteReadingList deletes a reading list owned by the authenticated user.
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
func (h *ReadingListHandler) listReadingListBooks(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	limit, offset := parseLimitOffset(r, defaultPageLimit, maxPageLimit)

	books, total, err := h.DB.ListReadingListBooks(ctx, listID, userID, limit, offset)
	if handleReadingListOpErr(ctx, w, err, "failed to list reading list books",
		slog.String(otelkeys.ReadingListID, listID),
	) {
		return
	}

	writeJSON(ctx, w, http.StatusOK, bookListDTO{
		Books:  mapSlice(books, toBookSummaryDTO),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// addBookToReadingList adds a book to a reading list (idempotent).
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
	if handleReadingListOpErr(ctx, w, err, "failed to add book to reading list",
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
func (h *ReadingListHandler) removeBookFromReadingList(w http.ResponseWriter, r *http.Request, listID, bookID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	removed, err := h.DB.RemoveBookFromReadingList(ctx, listID, userID, bookID)
	if handleReadingListOpErr(ctx, w, err, "failed to remove book from reading list",
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
