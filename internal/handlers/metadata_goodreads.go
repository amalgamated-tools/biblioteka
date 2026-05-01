package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/hibiken/asynq"
)

// getPendingMetadataOrErr fetches the pending metadata for a book and writes
// the appropriate error response if missing or unavailable. Returns the
// metadata and true on success; returns nil and false when it wrote an error
// response (caller should return).
func getPendingMetadataOrErr(ctx context.Context, d *db.DB, w http.ResponseWriter, userID, bookID string) (*db.GoodreadsMetadata, bool) {
	gm, err := d.GetPendingGoodreadsMetadataByBook(ctx, userID, bookID)
	if handleDBErr(ctx, w, err, "pending Goodreads metadata") {
		return nil, false
	}
	return gm, true
}

// getPendingMetadata returns the most recent pending metadata for a book.
//
//	@Summary		Get pending book metadata
//	@Description	Returns the most recent pending Goodreads/Hardcover metadata for a book
//	@Tags			Book Metadata
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{object}	metadataDTO
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/metadata [get]
func (h *MetadataHandler) getPendingMetadata(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	gm, ok := getPendingMetadataOrErr(r.Context(), h.DB, w, userID, bookID)
	if !ok {
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, toMetadataDTO(gm))
}

// fetchMetadata enqueues a metadata enrichment job for the given book.
//
//	@Summary		Fetch book metadata
//	@Description	Enqueue a background job to fetch book metadata from Goodreads/Hardcover. Returns 202 if already running or already fetched.
//	@Tags			Book Metadata
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		202	{object}	fetchMetadataResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/books/{id}/metadata/fetch [post]
func (h *MetadataHandler) fetchMetadata(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	if h.Enqueuer == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "background worker not available")
		return
	}

	// Verify the book exists.
	_, err := h.DB.GetBook(r.Context(), bookID)
	if handleDBErr(r.Context(), w, err, "book") {
		return
	}

	// If a pending metadata record already exists, short-circuit with 202 so
	// the client can listen on SSE or poll without triggering a duplicate job.
	existing, lookupErr := h.DB.GetPendingGoodreadsMetadataByBook(r.Context(), userID, bookID)
	if lookupErr == nil {
		slog.DebugContext(r.Context(), "pending metadata already exists, skipping enqueue",
			slog.String(otelkeys.BookID, bookID),
			slog.String(otelkeys.GoodreadsMetadataID, existing.ID),
		)
		writeJSON(r.Context(), w, http.StatusAccepted, fetchMetadataResponse{TaskID: "", Status: "already_exists"})
		return
	} else if !errors.Is(lookupErr, sql.ErrNoRows) {
		slog.ErrorContext(r.Context(), "failed to check pending metadata",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, lookupErr),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to check pending metadata")
		return
	}

	taskID, err := h.Enqueuer.Enqueue(r.Context(), jobs.JobEnrichGoodreads, jobs.EnrichGoodreadsPayload{
		BookID: bookID,
		UserID: userID,
	})
	if err != nil {
		// asynq returns ErrDuplicateTask when a unique task is already enqueued —
		// treat as a successful 202 since the job is in-flight.
		if errors.Is(err, asynq.ErrDuplicateTask) {
			writeJSON(r.Context(), w, http.StatusAccepted, fetchMetadataResponse{TaskID: "", Status: "already_running"})
			return
		}
		slog.ErrorContext(r.Context(), "failed to enqueue metadata fetch",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to enqueue metadata fetch")
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionMetadataFetchRequested, "book", bookID, map[string]any{"task_id": taskID})

	writeJSON(r.Context(), w, http.StatusAccepted, fetchMetadataResponse{TaskID: taskID, Status: "enqueued"})
}

// applyMetadata applies the pending metadata to the book and marks it as applied.
// The frontend UI currently uses a field-by-field workflow (copy into form, save
// via UpdateBook, then reject the pending record), so this endpoint is primarily
// intended for programmatic/CLI consumers that want a one-shot apply.
//
//	@Summary		Apply pending metadata to a book
//	@Description	Applies the pending Goodreads/Hardcover metadata to the book and marks it as applied. Primarily for programmatic/CLI consumers; the frontend uses a field-by-field workflow.
//	@Tags			Book Metadata
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{object}	bookSummaryDTO
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/metadata/apply [post]
func (h *MetadataHandler) applyMetadata(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	gm, ok := getPendingMetadataOrErr(r.Context(), h.DB, w, userID, bookID)
	if !ok {
		return
	}

	// Build a BookInput from the metadata, falling back to current book values.
	book, err := h.DB.GetBook(r.Context(), bookID)
	if handleDBErr(r.Context(), w, err, "book") {
		return
	}

	input := db.BookInput{
		Title: coalesceStr(gm.Title, book.Title),
		// Note: gm.AuthorName is intentionally not applied here. Authors are a
		// separate many-to-many relationship managed via SetBookAuthors.
		Description:     coalescePtr(gm.Description, book.Description),
		ASIN:            coalescePtr(gm.ASIN, book.ASIN),
		ISBN10:          coalescePtr(gm.ISBN10, book.ISBN10),
		ISBN13:          coalescePtr(gm.ISBN13, book.ISBN13),
		GoodreadsID:     coalescePtr(gm.GoodreadsID, book.GoodreadsID),
		HardcoverID:     coalescePtr(gm.HardcoverID, book.HardcoverID),
		GoogleBooksID:   coalescePtr(gm.GoogleBooksID, book.GoogleBooksID),
		PublicationDate: coalescePtr(gm.PublicationDate, book.PublicationDate),
		Publisher:       coalescePtr(gm.Publisher, book.Publisher),
		Language:        coalescePtr(gm.Language, book.Language),
		CoverImageURL:   coalescePtr(gm.CoverImageURL, book.CoverImageURL),
	}

	updated, err := h.DB.UpdateBook(r.Context(), bookID, input)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to apply metadata to book",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to apply metadata")
		return
	}

	// Mark the metadata as applied. If this fails, roll back is not feasible
	// (the book is already updated), but we must tell the caller so the pending
	// record can be cleaned up manually rather than silently reappearing.
	if _, err := h.DB.UpdateGoodreadsMetadataStatus(r.Context(), userID, gm.ID, db.GoodreadsMetadataStatusApplied); err != nil {
		slog.ErrorContext(r.Context(), "failed to mark metadata as applied",
			slog.String(otelkeys.GoodreadsMetadataID, gm.ID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "book updated but failed to mark metadata as applied")
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionMetadataApplied, "book", bookID,
		map[string]any{"metadata_id": gm.ID, "source": db.MetadataSourceGoodreads})

	writeJSON(r.Context(), w, http.StatusOK, toBookSummaryDTO(updated))
}

// rejectMetadata marks the pending metadata as rejected.
//
//	@Summary		Reject pending metadata
//	@Description	Marks the pending Goodreads/Hardcover metadata for a book as rejected
//	@Tags			Book Metadata
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		204	"No Content"
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/metadata/reject [post]
func (h *MetadataHandler) rejectMetadata(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	gm, ok := getPendingMetadataOrErr(r.Context(), h.DB, w, userID, bookID)
	if !ok {
		return
	}

	if _, err := h.DB.UpdateGoodreadsMetadataStatus(r.Context(), userID, gm.ID, db.GoodreadsMetadataStatusRejected); err != nil {
		slog.ErrorContext(r.Context(), "failed to reject metadata",
			slog.String(otelkeys.GoodreadsMetadataID, gm.ID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to reject metadata")
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionMetadataRejected, "book", bookID,
		map[string]any{"metadata_id": gm.ID, "source": db.MetadataSourceGoodreads})

	w.WriteHeader(http.StatusNoContent)
}
