package handlers

import (
	"context"
	"log/slog"
	"net/http"

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
		slog.ErrorContext(ctx, "failed to get "+resourceName, slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to get "+resourceName)
		return
	}
	writeJSON(ctx, w, http.StatusOK, mapSlice(items, toDTO))
}

// putBookSubResource is a generic helper for the decode-then-set-then-respond
// pattern common to PUT handlers for book sub-resources. It decodes the JSON
// request body into Req, extracts the payload via extractPayload, calls setFn
// to persist the change, and then re-fetches and writes the updated resource.
func putBookSubResource[T any, DTO any, Req any, Payload any](
	w http.ResponseWriter,
	r *http.Request,
	bookID string,
	getFn func(context.Context, string) ([]T, error),
	setFn func(context.Context, string, Payload) error,
	extractPayload func(*Req) Payload,
	toDTO func(*T) DTO,
	resourceName string,
) {
	var req Req
	if !decodeJSON(r, w, &req) {
		return
	}
	if err := setFn(r.Context(), bookID, extractPayload(&req)); err != nil {
		slog.ErrorContext(r.Context(), "failed to set "+resourceName, slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to set "+resourceName)
		return
	}
	respondBookSubResource(r.Context(), w, bookID, getFn, toDTO, resourceName)
}
