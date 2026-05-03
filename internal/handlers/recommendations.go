package handlers

import (
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

const (
	defaultRecommendationLimit = 10
	maxRecommendationLimit     = 50
)

// RecommendationHandler holds dependencies for the recommendations endpoint.
type RecommendationHandler struct {
	DB *db.DB
}

// HandleRecommendations handles GET /api/recommendations and returns a scored
// list of books the user has not yet read, based on
// author overlap, series continuation, publisher match, and download popularity.
// Optional query parameter: limit (default 10, max 50).
//
//	@Summary		Get book recommendations
//	@Description	Returns a scored list of recommended books for the authenticated user based on reading history.
//	@Tags			Recommendations
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Max items to return (default 10, max 50)"
//	@Success		200		{array}		bookSummaryDTO
//	@Failure		401		{object}	errorResponse
//	@Failure		405		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/recommendations [get]
func (h *RecommendationHandler) HandleRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit := parseLimit(r, defaultRecommendationLimit, maxRecommendationLimit)

	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	slog.DebugContext(ctx, "fetching recommendations",
		slog.String(otelkeys.UserID, userID),
		slog.Int(otelkeys.Limit, limit),
	)

	books, err := h.DB.GetRecommendations(ctx, userID, limit)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch recommendations",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to fetch recommendations")
		return
	}

	writeJSON(ctx, w, http.StatusOK, mapSlice(books, toBookSummaryDTO))
}
