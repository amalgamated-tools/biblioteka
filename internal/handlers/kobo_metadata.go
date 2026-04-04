package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/kobo"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// HandleBookMetadata handles GET /v1/library/{uuid}/metadata.
func (h *KoboHandler) HandleBookMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeKoboJSON(w, http.StatusOK, []any{})
		return
	}
	tokenValue := auth.KoboTokenFromContext(r.Context())
	bookID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/library/"), "/metadata")
	if bookID == "" {
		writeKoboJSON(w, http.StatusNotFound, map[string]any{})
		return
	}

	book, err := h.DB.GetBook(r.Context(), bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeKoboJSON(w, http.StatusNotFound, map[string]any{})
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch book for kobo metadata", slog.Any(otelkeys.Error, err))
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	authors, err := h.DB.GetBookAuthors(r.Context(), bookID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to fetch authors for kobo metadata",
			slog.String(otelkeys.ID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}
	files, err := h.DB.ListBookFiles(r.Context(), bookID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to fetch files for kobo metadata",
			slog.String(otelkeys.ID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}
	series, err := h.DB.GetBookSeries(r.Context(), bookID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to fetch series for kobo metadata",
			slog.String(otelkeys.ID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	base := schemeAndHost(r)
	downloadURLs := kobo.DownloadURLs(base, tokenValue, bookID, files)
	writeKoboJSON(w, http.StatusOK, []any{kobo.BookMetadata(book, authors, series, downloadURLs)})
}
