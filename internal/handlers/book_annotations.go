package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// BookAnnotationHandler holds dependencies for book annotation endpoints.
type BookAnnotationHandler struct {
	DB *db.DB
}

// annotationDTO is the wire representation of a single book annotation.
type annotationDTO struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	BookID    string       `json:"book_id"`
	Text      string       `json:"text"`
	CFI       *string      `json:"cfi,omitempty"`
	GroupID   *string      `json:"group_id,omitempty"`
	UserName  string       `json:"user_name"`
	CreatedAt db.Timestamp `json:"created_at"`
	UpdatedAt db.Timestamp `json:"updated_at"`
}

// createAnnotationRequest is the request body for creating a book annotation.
type createAnnotationRequest struct {
	Text    string  `json:"text"`
	CFI     *string `json:"cfi"`
	GroupID *string `json:"group_id"`
}

// updateAnnotationRequest is the request body for updating a book annotation.
type updateAnnotationRequest struct {
	Text    string  `json:"text"`
	CFI     *string `json:"cfi"`
	GroupID *string `json:"group_id"`
}

func toAnnotationDTO(a *db.BookAnnotation) annotationDTO {
	return annotationDTO{
		ID:        a.ID,
		UserID:    a.UserID,
		BookID:    a.BookID,
		Text:      a.Text,
		CFI:       a.CFI,
		GroupID:   a.GroupID,
		UserName:  a.UserName,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

// HandleBookAnnotations dispatches GET and POST requests for
// /api/books/{id}/annotations.
func (h *BookAnnotationHandler) HandleBookAnnotations(w http.ResponseWriter, r *http.Request, bookID string) {
	switch r.Method {
	case http.MethodGet:
		h.handleListBookAnnotations(w, r, bookID)
	case http.MethodPost:
		h.handleCreateBookAnnotation(w, r, bookID)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleAnnotation dispatches GET, PUT, and DELETE requests for
// /api/annotations/{id}.
func (h *BookAnnotationHandler) HandleAnnotation(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/annotations/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid annotation ID")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleGetAnnotation(w, r, id)
	case http.MethodPut:
		h.handleUpdateAnnotation(w, r, id)
	case http.MethodDelete:
		h.handleDeleteAnnotation(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleListBookAnnotations handles GET /api/books/{id}/annotations.
// Note: this returns both own and group-shared annotations, so a client may
// receive IDs it does not own. GET /api/annotations/{id} (GetAnnotation)
// filters strictly by user_id and will return 404 for group-shared IDs.
// This asymmetry is intentional: the list shows shared context; individual
// access is restricted to the annotation owner.
func (h *BookAnnotationHandler) handleListBookAnnotations(w http.ResponseWriter, r *http.Request, bookID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	annotations, err := h.DB.ListAnnotationsForBook(ctx, bookID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list book annotations",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to list book annotations")
		return
	}

	writeJSON(ctx, w, http.StatusOK, mapSlice(annotations, toAnnotationDTO))
}

func (h *BookAnnotationHandler) handleCreateBookAnnotation(w http.ResponseWriter, r *http.Request, bookID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	var req createAnnotationRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		writeError(ctx, w, http.StatusBadRequest, "text is required")
		return
	}

	annotation, err := h.DB.CreateAnnotation(ctx, userID, bookID, req.Text, req.CFI, req.GroupID)
	if err != nil {
		if errors.Is(err, db.ErrNotGroupMember) {
			writeError(ctx, w, http.StatusForbidden, "not a member of the specified group")
			return
		}
		slog.ErrorContext(ctx, "failed to create annotation",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to create annotation")
		return
	}

	logAudit(ctx, h.DB, userID, db.AuditActionAnnotationCreated, "annotation", annotation.ID, nil)
	writeJSON(ctx, w, http.StatusCreated, toAnnotationDTO(annotation))
}

func (h *BookAnnotationHandler) handleGetAnnotation(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	annotation, err := h.DB.GetAnnotation(ctx, id, userID)
	if handleDBErr(ctx, w, err, "annotation") {
		return
	}

	writeJSON(ctx, w, http.StatusOK, toAnnotationDTO(annotation))
}

func (h *BookAnnotationHandler) handleUpdateAnnotation(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	var req updateAnnotationRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		writeError(ctx, w, http.StatusBadRequest, "text is required")
		return
	}

	annotation, err := h.DB.UpdateAnnotation(ctx, id, userID, req.Text, req.CFI, req.GroupID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(ctx, w, http.StatusNotFound, "annotation not found")
			return
		}
		if errors.Is(err, db.ErrNotGroupMember) {
			writeError(ctx, w, http.StatusForbidden, "not a member of the specified group")
			return
		}
		slog.ErrorContext(ctx, "failed to update annotation",
			slog.String(otelkeys.AnnotationID, id),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to update annotation")
		return
	}

	logAudit(ctx, h.DB, userID, db.AuditActionAnnotationUpdated, "annotation", annotation.ID, nil)
	writeJSON(ctx, w, http.StatusOK, toAnnotationDTO(annotation))
}

func (h *BookAnnotationHandler) handleDeleteAnnotation(w http.ResponseWriter, r *http.Request, id string) {
	deleteUserOwnedResource(h.DB, w, r, id, "annotation", "annotation", otelkeys.AnnotationID,
		h.DB.GetAnnotation, h.DB.DeleteAnnotation,
		db.AuditActionAnnotationDeleted,
		nil,
	)
}
