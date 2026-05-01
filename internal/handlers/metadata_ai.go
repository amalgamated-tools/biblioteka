package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
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

func getPendingAIEnrichmentOrErr(ctx context.Context, d *db.DB, w http.ResponseWriter, userID, bookID string) (*db.AIEnrichment, bool) {
	return getPendingOrErr(ctx, w,
		func(ctx context.Context) (*db.AIEnrichment, error) {
			return d.GetPendingAIEnrichmentByBook(ctx, userID, bookID)
		},
		"pending AI enrichment",
	)
}

// fetchAIEnrichment enqueues an AI enrichment job for the given book.
//
//	@Summary		Fetch AI enrichment
//	@Description	Enqueue a background job to generate AI enrichment (tags, description, reading level) for a book. Returns 202 if already running or already fetched.
//	@Tags			Book Metadata
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		202	{object}	fetchMetadataResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/books/{id}/metadata/ai-fetch [post]
func (h *MetadataHandler) fetchAIEnrichment(w http.ResponseWriter, r *http.Request, bookID string) {
	if h.LLMProvider == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "LLM provider not configured")
		return
	}

	if h.Enqueuer == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "background worker not available")
		return
	}

	h.enqueueEnrichmentJob(w, r, bookID, enrichmentEnqueueConfig{
		jobType: jobs.JobEnrichAI,
		buildPayload: func(bookID, userID string) any {
			return jobs.EnrichAIPayload{BookID: bookID, UserID: userID}
		},
		getPendingID: func(ctx context.Context, userID, bookID string) (string, error) {
			e, err := h.DB.GetPendingAIEnrichmentByBook(ctx, userID, bookID)
			if err != nil {
				return "", err
			}
			return e.ID, nil
		},
		auditAction:   db.AuditActionAIEnrichmentFetchRequested,
		resourceLabel: "AI enrichment",
		idLogAttr: func(id string) slog.Attr {
			return slog.String(otelkeys.AIEnrichmentID, id)
		},
	})
}

// getPendingAIEnrichment returns the most recent pending AI enrichment for a book.
//
//	@Summary		Get pending AI enrichment
//	@Description	Returns the most recent pending AI enrichment for a book
//	@Tags			Book Metadata
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{object}	aiEnrichmentDTO
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/metadata/ai [get]
func (h *MetadataHandler) getPendingAIEnrichment(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	enrichment, ok := getPendingAIEnrichmentOrErr(r.Context(), h.DB, w, userID, bookID)
	if !ok {
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, toAIEnrichmentDTO(enrichment))
}

// applyAIEnrichment applies a pending AI enrichment to the book.
// It union-merges suggested tags and sets the description if the book has none.
// All mutations (tags, description, status) are applied atomically.
//
//	@Summary		Apply AI enrichment
//	@Description	Atomically applies a pending AI enrichment to the book: union-merges suggested tags and optionally sets the description (only if the book has none)
//	@Tags			Book Metadata
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{object}	aiEnrichmentDTO
//	@Failure		404	{object}	errorResponse
//	@Failure		409	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/metadata/ai-apply [post]
func (h *MetadataHandler) applyAIEnrichment(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	enrichment, ok := getPendingAIEnrichmentOrErr(r.Context(), h.DB, w, userID, bookID)
	if !ok {
		return
	}

	// Find or create tags for each suggested tag, skipping blank entries.
	newTagIDs := make([]string, 0, len(enrichment.SuggestedTags))
	for _, tagName := range enrichment.SuggestedTags {
		if strings.TrimSpace(tagName) == "" {
			continue
		}
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

	// Set description if book has none and enrichment has one.
	book, err := h.DB.GetBook(r.Context(), bookID)
	if handleDBErr(r.Context(), w, err, "book") {
		return
	}

	applyInput := db.ApplyAIEnrichmentInput{
		BookID:       bookID,
		UserID:       userID,
		EnrichmentID: enrichment.ID,
		NewTagIDs:    newTagIDs,
	}

	if (book.Description == nil || *book.Description == "") && enrichment.GeneratedDescription != nil && *enrichment.GeneratedDescription != "" {
		applyInput.Description = enrichment.GeneratedDescription
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
//
//	@Summary		Reject AI enrichment
//	@Description	Marks the pending AI enrichment for a book as rejected
//	@Tags			Book Metadata
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		204	"No Content"
//	@Failure		404	{object}	errorResponse
//	@Failure		409	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/metadata/ai-reject [post]
func (h *MetadataHandler) rejectAIEnrichment(w http.ResponseWriter, r *http.Request, bookID string) {
	userID := auth.UserIDFromContext(r.Context())

	enrichment, ok := getPendingAIEnrichmentOrErr(r.Context(), h.DB, w, userID, bookID)
	if !ok {
		return
	}

	if _, err := h.DB.UpdateAIEnrichmentStatus(r.Context(), userID, enrichment.ID, db.AIEnrichmentStatusRejected); err != nil {
		if errors.Is(err, db.ErrAIEnrichmentNotPending) {
			writeError(r.Context(), w, http.StatusConflict, "enrichment is no longer pending")
			return
		}
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
