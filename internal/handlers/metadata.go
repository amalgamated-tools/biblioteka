package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/pubsub"
	"github.com/hibiken/asynq"
)

// MetadataHandler handles metadata fetch, review, and apply endpoints for books.
type MetadataHandler struct {
	DB         *db.DB
	Enqueuer   jobs.Enqueuer
	Subscriber pubsub.Subscriber
}

// HandleBookMetadata dispatches /api/books/{id}/metadata and its sub-paths.
// It expects the bookID to have already been extracted from the URL path.
func (h *MetadataHandler) HandleBookMetadata(w http.ResponseWriter, r *http.Request, bookID string) {
	// Extract the action segment after /api/books/{id}/metadata/ using the
	// shared path helper. For bare /metadata, action will be "" (ok == false).
	metadataPrefix := "/api/books/" + bookID + "/metadata/"
	action, _, ok := extractPathSegments(r.URL.Path, metadataPrefix)
	if !ok {
		// No action segment — this is GET /api/books/{id}/metadata itself.
		action = ""
	}

	switch action {
	case "":
		if r.Method != http.MethodGet {
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.getPendingMetadata(w, r, bookID)
	case "fetch":
		if r.Method != http.MethodPost {
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.fetchMetadata(w, r, bookID)
	case "events":
		if r.Method != http.MethodGet {
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.streamEvents(w, r, bookID)
	case "apply":
		if r.Method != http.MethodPost {
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.applyMetadata(w, r, bookID)
	case "reject":
		if r.Method != http.MethodPost {
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.rejectMetadata(w, r, bookID)
	default:
		writeError(r.Context(), w, http.StatusNotFound, "not found")
	}
}

type metadataDTO struct {
	ID              string       `json:"id"`
	BookID          *string      `json:"book_id"`
	Status          string       `json:"status"`
	Source          string       `json:"source"`
	Title           *string      `json:"title"`
	Description     *string      `json:"description"`
	ASIN            *string      `json:"asin"`
	ISBN10          *string      `json:"isbn10"`
	ISBN13          *string      `json:"isbn13"`
	GoodreadsID     *string      `json:"goodreads_id"`
	HardcoverID     *string      `json:"hardcover_id"`
	GoogleBooksID   *string      `json:"google_books_id"`
	PublicationDate *string      `json:"publication_date"`
	Publisher       *string      `json:"publisher"`
	Language        *string      `json:"language"`
	CoverImageURL   *string      `json:"cover_image_url"`
	AuthorName      *string      `json:"author_name"`
	CreatedAt       db.Timestamp `json:"created_at"`
	UpdatedAt       db.Timestamp `json:"updated_at"`
}

// Intentionally omitted from the DTO: AuthorGoodreadsID, AuthorImageURL,
// GoodreadsWorkID, GoodreadsBookLegacyID, GoodreadsWorkLegacyID, and
// GoodreadsAuthorLegacyID. These are internal Goodreads identifiers used
// only during enrichment and have no value in the user-facing API.
func toMetadataDTO(gm *db.GoodreadsMetadata) metadataDTO {
	return metadataDTO{
		ID:              gm.ID,
		BookID:          gm.BookID,
		Status:          gm.Status,
		Source:          "goodreads",
		Title:           gm.Title,
		Description:     gm.Description,
		ASIN:            gm.ASIN,
		ISBN10:          gm.ISBN10,
		ISBN13:          gm.ISBN13,
		GoodreadsID:     gm.GoodreadsID,
		HardcoverID:     gm.HardcoverID,
		GoogleBooksID:   gm.GoogleBooksID,
		PublicationDate: gm.PublicationDate,
		Publisher:       gm.Publisher,
		Language:        gm.Language,
		CoverImageURL:   gm.CoverImageURL,
		AuthorName:      gm.AuthorName,
		CreatedAt:       gm.CreatedAt,
		UpdatedAt:       gm.UpdatedAt,
	}
}

// getPendingMetadataOrErr fetches the pending metadata for a book and writes
// the appropriate error response if missing or unavailable. Returns the
// metadata and true on success; returns nil and false when it wrote an error
// response (caller should return).
func getPendingMetadataOrErr(ctx context.Context, d *db.DB, w http.ResponseWriter, userID, bookID string) (*db.GoodreadsMetadata, bool) {
	gm, err := d.GetPendingGoodreadsMetadataByBook(ctx, userID, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(ctx, w, http.StatusNotFound, "no pending metadata found")
			return nil, false
		}
		slog.ErrorContext(ctx, "failed to get pending metadata",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to get pending metadata")
		return nil, false
	}
	return gm, true
}

// getPendingMetadata returns the most recent pending metadata for a book.
func (h *MetadataHandler) getPendingMetadata(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	gm, ok := getPendingMetadataOrErr(r.Context(), h.DB, w, userID, bookID)
	if !ok {
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, toMetadataDTO(gm))
}

type fetchMetadataResponse struct {
	TaskID string `json:"task_id,omitempty"`
	Status string `json:"status"`
}

// fetchMetadata enqueues a metadata enrichment job for the given book.
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

// sseWriteTimeout is the maximum time an SSE connection stays open.
const sseWriteTimeout = 2 * time.Minute

// sseHeartbeatInterval is the interval between SSE keepalive comments.
const sseHeartbeatInterval = 15 * time.Second

// streamEvents opens an SSE connection that streams metadata fetch progress
// events from Redis pub/sub.
func (h *MetadataHandler) streamEvents(w http.ResponseWriter, r *http.Request, bookID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(r.Context(), w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	if h.Subscriber == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "event streaming not available")
		return
	}

	// Verify the book exists before opening a long-lived SSE connection.
	if _, err := h.DB.GetBook(r.Context(), bookID); handleDBErr(r.Context(), w, err, "book") {
		return
	}

	// Extend the write deadline for this long-lived connection.
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout)); err != nil {
		slog.WarnContext(r.Context(), "failed to set write deadline for SSE",
			slog.Any(otelkeys.Error, err),
		)
	}

	// Subscribe to Redis pub/sub BEFORE flushing headers. The client opens
	// this SSE connection and then immediately POSTs /metadata/fetch, so the
	// subscription must be confirmed before the client can trigger the job —
	// otherwise events published between Flush() and Subscribe() are lost.
	userID := auth.UserIDFromContext(r.Context())
	channel := pubsub.MetadataChannel(bookID, userID)

	msgs, cancel := h.Subscriber.Subscribe(r.Context(), channel)
	defer cancel()

	// Now that the subscription is active, send SSE headers so the client
	// knows the stream is ready and can proceed to trigger the fetch job.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
	flusher.Flush()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			// SSE comment line as keepalive.
			if err := rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout)); err != nil {
				slog.WarnContext(r.Context(), "failed to extend write deadline for SSE heartbeat",
					slog.Any(otelkeys.Error, err),
				)
			}
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case msg, ok := <-msgs:
			if !ok {
				return
			}

			// Extend deadline on each write.
			if err := rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout)); err != nil {
				slog.WarnContext(r.Context(), "failed to extend write deadline for SSE",
					slog.Any(otelkeys.Error, err),
				)
			}

			_, err := fmt.Fprintf(w, "data: %s\n\n", msg)
			if err != nil {
				return
			}
			flusher.Flush()

			// Close the connection on terminal events.
			var evt struct {
				Event string `json:"event"`
			}
			if json.Unmarshal([]byte(msg), &evt) == nil {
				if evt.Event == jobs.EventComplete || evt.Event == jobs.EventError || evt.Event == jobs.EventNotFound {
					return
				}
			}
		}
	}
}

// applyMetadata applies the pending metadata to the book and marks it as applied.
// The frontend UI currently uses a field-by-field workflow (copy into form, save
// via UpdateBook, then reject the pending record), so this endpoint is primarily
// intended for programmatic/CLI consumers that want a one-shot apply.
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
		map[string]any{"metadata_id": gm.ID, "source": "goodreads"})

	writeJSON(r.Context(), w, http.StatusOK, toBookSummaryDTO(updated))
}

// rejectMetadata marks the pending metadata as rejected.
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
		map[string]any{"metadata_id": gm.ID, "source": "goodreads"})

	w.WriteHeader(http.StatusNoContent)
}

// coalesceStr returns the pointed-to string if non-nil and non-empty,
// otherwise falls back to the default value. An empty string is treated as
// absent so that metadata fields explicitly cleared by the provider do not
// overwrite existing book data with blanks.
func coalesceStr(ptr *string, fallback string) string {
	if ptr != nil && *ptr != "" {
		return *ptr
	}
	return fallback
}

// coalescePtr returns ptr if non-nil and points to a non-empty string,
// otherwise returns fallback. Like coalesceStr, empty strings are treated as
// absent to prevent blank metadata values from overwriting existing book data.
func coalescePtr(ptr, fallback *string) *string {
	if ptr != nil && *ptr != "" {
		return ptr
	}
	return fallback
}
