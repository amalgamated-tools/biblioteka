package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// HandleBookState handles GET/PUT /v1/library/{uuid}/state.
func (h *KoboHandler) HandleBookState(w http.ResponseWriter, r *http.Request) {
	bookID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/library/"), "/state")
	userID := auth.UserIDFromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		h.getBookState(w, r, userID, bookID)
	case http.MethodPut:
		h.updateBookState(w, r, userID, bookID)
	default:
		writeKoboJSON(w, http.StatusOK, []any{})
	}
}

func (h *KoboHandler) getBookState(w http.ResponseWriter, r *http.Request, userID, bookID string) {
	book, err := h.DB.GetBook(r.Context(), bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeKoboJSON(w, http.StatusNotFound, map[string]any{})
			return
		}
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	state, err := h.DB.GetKoboReadingState(r.Context(), userID, bookID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}
	if state == nil {
		// Return a default "ReadyToRead" state.
		state = &db.KoboReadingState{
			BookID:    bookID,
			Status:    "ReadyToRead",
			CreatedAt: book.CreatedAt,
			UpdatedAt: book.UpdatedAt,
		}
	}

	writeKoboJSON(w, http.StatusOK, []any{koboReadingStateResponse(state)})
}

type koboStateUpdateRequest struct {
	ReadingStates []struct {
		CurrentBookmark *struct {
			ProgressPercent              *float64 `json:"ProgressPercent"`
			ContentSourceProgressPercent *float64 `json:"ContentSourceProgressPercent"`
			Location                     *struct {
				Value  string `json:"Value"`
				Type   string `json:"Type"`
				Source string `json:"Source"`
			} `json:"Location"`
		} `json:"CurrentBookmark"`
		Statistics *struct {
			SpentReadingMinutes  *int `json:"SpentReadingMinutes"`
			RemainingTimeMinutes *int `json:"RemainingTimeMinutes"`
		} `json:"Statistics"`
		StatusInfo *struct {
			Status string `json:"Status"`
		} `json:"StatusInfo"`
	} `json:"ReadingStates"`
}

func (h *KoboHandler) updateBookState(w http.ResponseWriter, r *http.Request, userID, bookID string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req koboStateUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.ReadingStates) == 0 {
		writeKoboJSON(w, http.StatusBadRequest, map[string]any{"RequestResult": "BadRequest"})
		return
	}

	if _, err := h.DB.GetBook(r.Context(), bookID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeKoboJSON(w, http.StatusNotFound, map[string]any{"RequestResult": "NotFound"})
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch book for kobo state update",
			slog.String(otelkeys.ID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{"RequestResult": "ServerError"})
		return
	}

	rs := req.ReadingStates[0]

	status := "ReadyToRead"
	if rs.StatusInfo != nil && rs.StatusInfo.Status != "" {
		status = rs.StatusInfo.Status
	}

	var percentRead *float64
	var locationValue, locationType, locationSource *string
	if rs.CurrentBookmark != nil {
		percentRead = rs.CurrentBookmark.ProgressPercent
		if rs.CurrentBookmark.Location != nil {
			lv := rs.CurrentBookmark.Location.Value
			lt := rs.CurrentBookmark.Location.Type
			ls := rs.CurrentBookmark.Location.Source
			locationValue = &lv
			locationType = &lt
			locationSource = &ls
		}
	}

	state, err := h.DB.UpsertKoboReadingState(r.Context(), userID, bookID, status, percentRead, locationValue, locationType, locationSource)
	if err != nil {
		// Treat missing books (e.g., foreign key violations or not-found translations) as a 404-style response
		if errors.Is(err, sql.ErrNoRows) ||
			strings.Contains(err.Error(), "FOREIGN KEY constraint failed") ||
			strings.Contains(err.Error(), "violates foreign key constraint") {
			writeKoboJSON(w, http.StatusNotFound, map[string]any{"RequestResult": "NotFound"})
			return
		}

		slog.ErrorContext(r.Context(), "failed to upsert kobo reading state", slog.Any(otelkeys.Error, err))
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{"RequestResult": "ServerError"})
		return
	}

	updated := state.UpdatedAt.UTC().Format(time.RFC3339)
	writeKoboJSON(w, http.StatusOK, map[string]any{
		"RequestResult": "Success",
		"UpdateResults": []any{
			map[string]any{
				"EntitlementId":         bookID,
				"CurrentBookmarkResult": map[string]any{"Result": "Success"},
				"StatisticsResult":      map[string]any{"Result": "Success"},
				"StatusInfoResult":      map[string]any{"Result": "Success"},
				"LastModified":          updated,
				"PriorityTimestamp":     updated,
			},
		},
	})
}

func koboReadingStateResponse(state *db.KoboReadingState) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339)

	updatedAt := now
	if !state.UpdatedAt.IsZero() {
		updatedAt = state.UpdatedAt.UTC().Format(time.RFC3339)
	}
	createdAt := updatedAt
	if !state.CreatedAt.IsZero() {
		createdAt = state.CreatedAt.UTC().Format(time.RFC3339)
	}

	statusInfo := map[string]any{
		"LastModified":        updatedAt,
		"Status":              state.Status,
		"TimesStartedReading": 0,
	}

	currentBookmark := map[string]any{
		"LastModified": updatedAt,
	}
	if state.PercentRead != nil {
		currentBookmark["ProgressPercent"] = *state.PercentRead
		currentBookmark["ContentSourceProgressPercent"] = *state.PercentRead
	}
	if state.LocationValue != nil && state.LocationType != nil && state.LocationSource != nil {
		currentBookmark["Location"] = map[string]any{
			"Value":  *state.LocationValue,
			"Type":   *state.LocationType,
			"Source": *state.LocationSource,
		}
	}

	return map[string]any{
		"EntitlementId":     state.BookID,
		"Created":           createdAt,
		"LastModified":      updatedAt,
		"PriorityTimestamp": updatedAt,
		"StatusInfo":        statusInfo,
		"Statistics":        map[string]any{"LastModified": updatedAt},
		"CurrentBookmark":   currentBookmark,
	}
}
