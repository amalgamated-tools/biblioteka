package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

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
