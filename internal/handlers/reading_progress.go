package handlers

import (
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ReadingProgressHandler holds dependencies for reading progress endpoints.
type ReadingProgressHandler struct {
	DB *db.DB
}

// readingProgressItemDTO is the wire representation of a single in-progress
// document returned by the stats endpoint.
type readingProgressItemDTO struct {
	Document                  string  `json:"document"`
	Percentage                float64 `json:"percentage"`
	Device                    *string `json:"device,omitempty"`
	LastSynced                string  `json:"last_synced"`
	EstimatedMinutesRemaining *int64  `json:"estimated_minutes_remaining,omitempty"`
}

// readingProgressStatsResponse is the response body for GET /api/reading-progress/stats.
type readingProgressStatsResponse struct {
	CurrentStreak int                      `json:"current_streak"`
	TotalTracked  int                      `json:"total_tracked"`
	TotalFinished int                      `json:"total_finished"`
	InProgress    []readingProgressItemDTO `json:"in_progress"`
}

// HandleReadingProgressStats handles GET /api/reading-progress/stats.
// It returns the authenticated user's reading streak, overall counts, and
// a list of documents currently in progress (0 < percentage < 0.99).
//
//	@Summary		Get reading progress statistics
//	@Description	Returns the authenticated user's reading streak, total books tracked, finished count, and list of in-progress documents.
//	@Tags			reading-progress
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	readingProgressStatsResponse
//	@Failure		401	{object}	errorResponse	"Unauthorized"
//	@Failure		405	{object}	errorResponse	"Method not allowed"
//	@Failure		500	{object}	errorResponse	"Internal server error"
//	@Router			/reading-progress/stats [get]
func (h *ReadingProgressHandler) HandleReadingProgressStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	stats, err := h.DB.GetReadingStats(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get reading stats",
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.UserID, userID),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to get reading stats")
		return
	}

	streak, err := h.DB.GetReadingStreak(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get reading streak",
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.UserID, userID),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to get reading streak")
		return
	}

	progressList, err := h.DB.ListReadingProgress(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list reading progress",
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.UserID, userID),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to list reading progress")
		return
	}

	inProgress := make([]readingProgressItemDTO, 0)
	for i := range progressList {
		p := &progressList[i]
		if p.Percentage <= 0 || p.Percentage >= 0.99 {
			continue
		}
		inProgress = append(inProgress, toReadingProgressItemDTO(p))
	}

	writeJSON(ctx, w, http.StatusOK, readingProgressStatsResponse{
		CurrentStreak: streak,
		TotalTracked:  stats.TotalTracked,
		TotalFinished: stats.TotalFinished,
		InProgress:    inProgress,
	})
}

func toReadingProgressItemDTO(p *db.ReadingProgress) readingProgressItemDTO {
	return readingProgressItemDTO{
		Document:                  p.Document,
		Percentage:                p.Percentage,
		Device:                    p.Device,
		LastSynced:                p.UpdatedAt.UTC().Format(time.RFC3339),
		EstimatedMinutesRemaining: estimateMinutesRemaining(p),
	}
}

// estimateMinutesRemaining returns a rough estimate of the time remaining to
// finish reading a document, based on elapsed reading time and current
// progress percentage. Returns nil when the data is insufficient (percentage ≤
// 1% or less than 5 minutes of tracked elapsed time).
func estimateMinutesRemaining(p *db.ReadingProgress) *int64 {
	if p.Percentage <= 0.01 {
		return nil
	}
	elapsed := p.UpdatedAt.Sub(p.CreatedAt.Time)
	if elapsed < 5*time.Minute {
		return nil
	}
	elapsedMinutes := elapsed.Minutes()
	remaining := (1 - p.Percentage) / p.Percentage * elapsedMinutes
	v := int64(math.Round(remaining))
	return &v
}
