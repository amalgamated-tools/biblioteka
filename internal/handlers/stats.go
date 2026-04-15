package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

const (
	defaultStatMonths = 12
	maxStatMonths     = 24
)

// StatsHandler holds dependencies for statistics endpoints.
type StatsHandler struct {
	DB *db.DB
}

type monthlyDownloadsDTO struct {
	Month string `json:"month"` // "YYYY-MM"
	Count int    `json:"count"`
}

func toMonthlyDownloadsDTO(c *db.MonthlyDownloadCount) monthlyDownloadsDTO {
	return monthlyDownloadsDTO{Month: c.Month, Count: c.Count}
}

// HandleDownloadsPerMonth handles GET /api/stats/downloads-per-month.
// It returns monthly download counts for the authenticated user.
// Optional query parameter: months (default 12, max 24).
func (h *StatsHandler) HandleDownloadsPerMonth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	months := defaultStatMonths
	if s := r.URL.Query().Get("months"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 1 {
			v = defaultStatMonths
		} else if v > maxStatMonths {
			v = maxStatMonths
		}
		months = v
	}

	userID := auth.UserIDFromContext(r.Context())

	slog.DebugContext(r.Context(), "fetching monthly download stats",
		slog.String(otelkeys.UserID, userID),
		slog.Int(otelkeys.Count, months),
	)

	counts, err := h.DB.GetMonthlyDownloads(r.Context(), userID, months)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to fetch monthly download stats",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to fetch download stats")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, mapSlice(counts, toMonthlyDownloadsDTO))
}

// HandleYearInBooks handles GET /api/stats/year-in-books.
// It returns an annual reading summary for the authenticated user.
// Optional query parameter: year (default: current year).
//
//	@Summary		Get Year in Books statistics
//	@Description	Returns annual reading statistics: books finished, active reading days, longest streak, and total downloads.
//	@Tags			stats
//	@Security		BearerAuth
//	@Produce		json
//	@Param			year	query		int	false	"Calendar year (default: current year)"
//	@Success		200		{object}	db.YearInBooks
//	@Failure		400		{object}	errorResponse	"Bad request"
//	@Failure		401		{object}	errorResponse	"Unauthorized"
//	@Failure		405		{object}	errorResponse	"Method not allowed"
//	@Failure		500		{object}	errorResponse	"Internal server error"
//	@Router			/stats/year-in-books [get]
func (h *StatsHandler) HandleYearInBooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	now := time.Now().UTC()
	year := now.Year()
	if s := r.URL.Query().Get("year"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 1 || v > now.Year()+1 {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid year")
			return
		}
		year = v
	}

	userID := auth.UserIDFromContext(r.Context())

	slog.DebugContext(r.Context(), "fetching year-in-books stats",
		slog.String(otelkeys.UserID, userID),
		slog.Int(otelkeys.Year, year),
	)

	stats, err := h.DB.GetYearInBooks(r.Context(), userID, year)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to fetch year-in-books stats",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to fetch year-in-books stats")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, stats)
}
