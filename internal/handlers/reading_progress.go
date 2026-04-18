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

	progressList, err := h.DB.ListReadingProgress(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list reading progress",
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.UserID, userID),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to list reading progress")
		return
	}

	// Compute all aggregate values from the already-fetched list to avoid
	// additional DB round-trips. GetReadingStats issued a separate COUNT(*)
	// query over the same table and WHERE clause; we can derive the same
	// values in a single pass.
	timestamps := make([]time.Time, len(progressList))
	totalFinished := 0
	inProgress := make([]readingProgressItemDTO, 0)
	for i := range progressList {
		p := &progressList[i]
		timestamps[i] = p.UpdatedAt.Time
		if p.Percentage >= 0.99 {
			totalFinished++
		} else if p.Percentage > 0 {
			inProgress = append(inProgress, toReadingProgressItemDTO(p))
		}
	}
	streak := db.ComputeReadingStreak(timestamps, time.Now().UTC())

	writeJSON(ctx, w, http.StatusOK, readingProgressStatsResponse{
		CurrentStreak: streak,
		TotalTracked:  len(progressList),
		TotalFinished: totalFinished,
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

// maxEstimateElapsed is the maximum wall-clock elapsed time between first and
// last sync that we consider reliable for estimating remaining reading time.
// Beyond this threshold the estimate becomes misleading (e.g., a book synced
// once a month ago would imply months of remaining time).
const maxEstimateElapsed = 30 * 24 * time.Hour // 30 days

// estimateMinutesRemaining returns a rough estimate of the time remaining to
// finish reading a document. The estimate is based on the wall-clock time
// between the first sync (CreatedAt) and the most recent sync (UpdatedAt),
// extrapolated linearly from the current progress percentage. Returns nil when
// the data is insufficient (percentage <= 1%, less than 5 minutes of elapsed
// time, or elapsed time exceeds 30 days making the estimate unreliable).
func estimateMinutesRemaining(p *db.ReadingProgress) *int64 {
	if p.Percentage <= 0.01 {
		return nil
	}
	elapsed := p.UpdatedAt.Sub(p.CreatedAt.Time)
	if elapsed < 5*time.Minute {
		return nil
	}
	if elapsed > maxEstimateElapsed {
		return nil
	}
	elapsedMinutes := elapsed.Minutes()
	remaining := (1 - p.Percentage) / p.Percentage * elapsedMinutes
	v := int64(math.Round(remaining))
	return &v
}
