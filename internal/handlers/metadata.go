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
	"github.com/amalgamated-tools/biblioteka/internal/llm"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/pubsub"
)

// MetadataHandler handles metadata fetch, review, and apply endpoints for books.
type MetadataHandler struct {
	DB          *db.DB
	Enqueuer    jobs.Enqueuer
	Subscriber  pubsub.Subscriber
	LLMProvider llm.Provider
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
	case "ai-fetch":
		if r.Method != http.MethodPost {
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.fetchAIEnrichment(w, r, bookID)
	case "ai-apply":
		if r.Method != http.MethodPost {
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.applyAIEnrichment(w, r, bookID)
	case "ai-reject":
		if r.Method != http.MethodPost {
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.rejectAIEnrichment(w, r, bookID)
	case "ai":
		if r.Method != http.MethodGet {
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.getPendingAIEnrichment(w, r, bookID)
	default:
		writeError(r.Context(), w, http.StatusNotFound, "not found")
	}
}

// getPendingOrErr is a generic helper that fetches a pending enrichment record
// for a book and writes the appropriate error response if missing or
// unavailable. Returns the record and true on success; returns nil and false
// when it wrote an error response (caller should return).
func getPendingOrErr[T any](
	ctx context.Context,
	w http.ResponseWriter,
	fetch func(context.Context) (*T, error),
	resourceName string,
) (*T, bool) {
	record, err := fetch(ctx)
	if handleDBErr(ctx, w, err, resourceName) {
		return nil, false
	}
	return record, true
}

// enrichmentEnqueueConfig configures the shared enqueue flow used by both the
// Goodreads metadata and AI enrichment fetch endpoints.
type enrichmentEnqueueConfig struct {
	jobType       string
	buildPayload  func(bookID, userID string) any
	getPendingID  func(ctx context.Context, userID, bookID string) (string, error)
	auditAction   string
	resourceLabel string // e.g. "metadata", "AI enrichment"
	idLogAttr     func(id string) slog.Attr
}

// enqueueEnrichmentJob encapsulates the shared check-existing → enqueue →
// handle-duplicate flow for metadata and AI enrichment fetch endpoints.
// The caller is responsible for any pre-checks (e.g. Enqueuer or LLMProvider
// availability) before calling this helper.
func (h *MetadataHandler) enqueueEnrichmentJob(w http.ResponseWriter, r *http.Request, bookID string, cfg enrichmentEnqueueConfig) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	// Verify the book exists.
	if _, err := h.DB.GetBook(ctx, bookID); handleDBErr(ctx, w, err, "book") {
		return
	}

	// If a pending record already exists, short-circuit with 202 so the client
	// can listen on SSE or poll without triggering a duplicate job.
	existingID, lookupErr := cfg.getPendingID(ctx, userID, bookID)
	if lookupErr == nil {
		slog.DebugContext(ctx, "pending "+cfg.resourceLabel+" already exists, skipping enqueue",
			slog.String(otelkeys.BookID, bookID),
			cfg.idLogAttr(existingID),
		)
		writeJSON(ctx, w, http.StatusAccepted, fetchMetadataResponse{Status: metadataStatusAlreadyExists})
		return
	} else if !errors.Is(lookupErr, sql.ErrNoRows) {
		slog.ErrorContext(ctx, "failed to check pending "+cfg.resourceLabel,
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, lookupErr),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to check pending "+cfg.resourceLabel)
		return
	}

	taskID, err := h.Enqueuer.Enqueue(ctx, cfg.jobType, cfg.buildPayload(bookID, userID))
	if err != nil {
		slog.ErrorContext(ctx, "failed to enqueue "+cfg.resourceLabel+" fetch",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to enqueue "+cfg.resourceLabel+" fetch")
		return
	}

	logAudit(ctx, h.DB, userID, cfg.auditAction, "book", bookID, map[string]any{"task_id": taskID})

	writeJSON(ctx, w, http.StatusAccepted, fetchMetadataResponse{TaskID: taskID, Status: metadataStatusEnqueued})
}
