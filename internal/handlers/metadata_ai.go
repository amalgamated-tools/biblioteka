package handlers

import (
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

type aiEnrichmentDTO struct {
	ID                   string       `json:"id"`
	BookID               *string      `json:"book_id"`
	Status               string       `json:"status"`
	Provider             string       `json:"provider"`
	Model                string       `json:"model"`
	SuggestedTags        []string     `json:"suggested_tags"`
	ReadingLevel         *string      `json:"reading_level"`
	GeneratedDescription *string      `json:"generated_description"`
	CreatedAt            db.Timestamp `json:"created_at"`
	UpdatedAt            db.Timestamp `json:"updated_at"`
}

func toAIEnrichmentDTO(e *db.AIEnrichment) aiEnrichmentDTO {
	tags := e.SuggestedTags
	if tags == nil {
		tags = []string{}
	}
	return aiEnrichmentDTO{
		ID:                   e.ID,
		BookID:               e.BookID,
		Status:               e.Status,
		Provider:             e.Provider,
		Model:                e.Model,
		SuggestedTags:        tags,
		ReadingLevel:         e.ReadingLevel,
		GeneratedDescription: e.GeneratedDescription,
		CreatedAt:            e.CreatedAt,
		UpdatedAt:            e.UpdatedAt,
	}
}

// fetchAIEnrichment enqueues an AI enrichment job for the given book.
func (h *MetadataHandler) fetchAIEnrichment(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	if h.LLMProvider == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "LLM provider not configured")
		return
	}

	if h.Enqueuer == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "background worker not available")
		return
	}

	// Verify the book exists.
	if _, err := h.DB.GetBook(r.Context(), bookID); handleDBErr(r.Context(), w, err, "book") {
		return
	}

	// If a pending AI enrichment already exists, short-circuit.
	existing, lookupErr := h.DB.GetPendingAIEnrichmentByBook(r.Context(), userID, bookID)
	if lookupErr == nil {
		slog.DebugContext(r.Context(), "pending AI enrichment already exists, skipping enqueue",
			slog.String(otelkeys.BookID, bookID),
			slog.String(otelkeys.AIEnrichmentID, existing.ID),
		)
		writeJSON(r.Context(), w, http.StatusAccepted, fetchMetadataResponse{Status: "already_exists"})
		return
	} else if !errors.Is(lookupErr, sql.ErrNoRows) {
		slog.ErrorContext(r.Context(), "failed to check pending AI enrichment",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, lookupErr),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to check pending AI enrichment")
		return
	}

	taskID, err := h.Enqueuer.Enqueue(r.Context(), jobs.JobEnrichAI, jobs.EnrichAIPayload{
		BookID: bookID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, asynq.ErrDuplicateTask) {
			writeJSON(r.Context(), w, http.StatusAccepted, fetchMetadataResponse{Status: "already_running"})
			return
		}
		slog.ErrorContext(r.Context(), "failed to enqueue AI enrichment fetch",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to enqueue AI enrichment fetch")
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionAIEnrichmentFetchRequested, "book", bookID, map[string]any{"task_id": taskID})

	writeJSON(r.Context(), w, http.StatusAccepted, fetchMetadataResponse{TaskID: taskID, Status: "enqueued"})
}

// getPendingAIEnrichment returns the most recent pending AI enrichment for a book.
func (h *MetadataHandler) getPendingAIEnrichment(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	enrichment, err := h.DB.GetPendingAIEnrichmentByBook(r.Context(), userID, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusNotFound, "no pending AI enrichment found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get pending AI enrichment",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get pending AI enrichment")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, toAIEnrichmentDTO(enrichment))
}

// applyAIEnrichment applies a pending AI enrichment to the book.
// It union-merges suggested tags and sets the description if the book has none.
// All mutations (tags, description, status) are applied atomically.
func (h *MetadataHandler) applyAIEnrichment(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	enrichment, err := h.DB.GetPendingAIEnrichmentByBook(r.Context(), userID, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusNotFound, "no pending AI enrichment found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get pending AI enrichment",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get pending AI enrichment")
		return
	}

	// Find or create tags for each suggested tag.
	newTagIDs := make([]string, 0, len(enrichment.SuggestedTags))
	for _, tagName := range enrichment.SuggestedTags {
		tag, err := h.DB.FindOrCreateTag(r.Context(), tagName)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to find or create tag",
				slog.String(otelkeys.Name, tagName),
				slog.Any(otelkeys.Error, err),
			)
			writeError(r.Context(), w, http.StatusInternalServerError, "failed to process tag: "+tagName)
			return
		}
		newTagIDs = append(newTagIDs, tag.ID)
	}

	// Get current book tags and union with new ones.
	currentTags, err := h.DB.GetBookTags(r.Context(), bookID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get current book tags",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get current book tags")
		return
	}

	tagIDSet := make(map[string]struct{})
	for _, t := range currentTags {
		tagIDSet[t.ID] = struct{}{}
	}
	for _, id := range newTagIDs {
		tagIDSet[id] = struct{}{}
	}
	unionIDs := make([]string, 0, len(tagIDSet))
	for id := range tagIDSet {
		unionIDs = append(unionIDs, id)
	}

	// Set description if book has none and enrichment has one.
	book, err := h.DB.GetBook(r.Context(), bookID)
	if handleDBErr(r.Context(), w, err, "book") {
		return
	}

	applyInput := db.ApplyAIEnrichmentInput{
		BookID:       bookID,
		UserID:       userID,
		EnrichmentID: enrichment.ID,
		TagIDs:       unionIDs,
	}

	if (book.Description == nil || *book.Description == "") && enrichment.GeneratedDescription != nil && *enrichment.GeneratedDescription != "" {
		applyInput.BookUpdate = &db.BookInput{
			Title:           book.Title,
			Description:     enrichment.GeneratedDescription,
			ASIN:            book.ASIN,
			ISBN10:          book.ISBN10,
			ISBN13:          book.ISBN13,
			GoodreadsID:     book.GoodreadsID,
			HardcoverID:     book.HardcoverID,
			GoogleBooksID:   book.GoogleBooksID,
			PublicationDate: book.PublicationDate,
			Publisher:       book.Publisher,
			Language:        book.Language,
			CoverImageURL:   book.CoverImageURL,
		}
	}

	updated, err := h.DB.ApplyAIEnrichment(r.Context(), applyInput)
	if err != nil {
		if errors.Is(err, db.ErrAIEnrichmentNotPending) {
			writeError(r.Context(), w, http.StatusConflict, "enrichment is no longer pending")
			return
		}
		slog.ErrorContext(r.Context(), "failed to apply AI enrichment",
			slog.String(otelkeys.AIEnrichmentID, enrichment.ID),
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to apply AI enrichment")
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionAIEnrichmentApplied, "book", bookID,
		map[string]any{"enrichment_id": enrichment.ID})

	writeJSON(r.Context(), w, http.StatusOK, toAIEnrichmentDTO(updated))
}

// rejectAIEnrichment marks the pending AI enrichment as rejected.
func (h *MetadataHandler) rejectAIEnrichment(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	enrichment, err := h.DB.GetPendingAIEnrichmentByBook(r.Context(), userID, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusNotFound, "no pending AI enrichment found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get pending AI enrichment",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get pending AI enrichment")
		return
	}

	if _, err := h.DB.UpdateAIEnrichmentStatus(r.Context(), userID, enrichment.ID, db.AIEnrichmentStatusRejected); err != nil {
		slog.ErrorContext(r.Context(), "failed to reject AI enrichment",
			slog.String(otelkeys.AIEnrichmentID, enrichment.ID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to reject AI enrichment")
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionAIEnrichmentRejected, "book", bookID,
		map[string]any{"enrichment_id": enrichment.ID})

	w.WriteHeader(http.StatusNoContent)
}
