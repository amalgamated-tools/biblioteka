package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// respondBookSubResource is a generic helper that fetches a book sub-resource
// by bookID, converts each item to a DTO, and writes the result as a JSON
// response. It is shared by the GET handler and the respond-after-set step in
// the PUT handler.
func respondBookSubResource[T any, DTO any](
	ctx context.Context,
	w http.ResponseWriter,
	bookID string,
	getFn func(context.Context, string) ([]T, error),
	toDTO func(*T) DTO,
	resourceName string,
) {
	items, err := getFn(ctx, bookID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get book sub-resource",
			slog.String(otelkeys.Resource, resourceName),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to get "+resourceName)
		return
	}
	writeJSON(ctx, w, http.StatusOK, mapSlice(items, toDTO))
}

// putBookSubResource is a generic helper for the decode-then-set-then-respond
// pattern common to PUT handlers for book sub-resources. It decodes the JSON
// request body into Req, extracts the payload via extractPayload, calls setFn
// to persist the change, and then re-fetches and writes the updated resource.
// When d and auditAction are both non-nil/non-empty, an audit log entry is
// written after a successful set.
func putBookSubResource[T any, DTO any, Req any, Payload any](
	w http.ResponseWriter,
	r *http.Request,
	bookID string,
	getFn func(context.Context, string) ([]T, error),
	setFn func(context.Context, string, Payload) error,
	extractPayload func(*Req) Payload,
	toDTO func(*T) DTO,
	resourceName string,
	d *db.DB,
	auditAction string,
) {
	ctx := r.Context()
	var req Req
	if !decodeJSON(r, w, &req) {
		return
	}
	if err := setFn(ctx, bookID, extractPayload(&req)); err != nil {
		slog.ErrorContext(ctx, "failed to set book sub-resource",
			slog.String(otelkeys.Resource, resourceName),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to set "+resourceName)
		return
	}
	if d != nil && auditAction != "" {
		userID := auth.UserIDFromContext(ctx)
		logAudit(ctx, d, userID, auditAction, "book", bookID, nil)
	}
	respondBookSubResource(ctx, w, bookID, getFn, toDTO, resourceName)
}

// listParentBooks is a generic helper for GET /{parent}/{id}/books endpoints.
// It fetches a paginated list of books belonging to the given parent entity,
// handles errors, and writes the JSON response. If no books are found it
// verifies the parent exists (returning 404 if not) before writing an empty
// result. idAttr is a slog attribute identifying the parent (e.g.
// slog.String(otelkeys.AuthorID, authorID)). parentLabel is used in error
// response messages (e.g. "author" or "series").
func listParentBooks[Parent any](
	w http.ResponseWriter,
	r *http.Request,
	parentID string,
	idAttr slog.Attr,
	listFn func(context.Context, string, int, int) ([]db.Book, int, error),
	getFn func(context.Context, string) (*Parent, error),
	parentLabel string,
) {
	ctx := r.Context()
	limit, offset := parseLimitOffset(r, defaultPageLimit, maxPageLimit)

	books, total, err := listFn(ctx, parentID, limit, offset)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to list parent books",
			idAttr,
			slog.String(otelkeys.Resource, parentLabel),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to list "+parentLabel+" books")
		return
	}

	if total == 0 {
		if _, err := getFn(ctx, parentID); handleDBErr(ctx, w, err, parentLabel) {
			return
		}
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
