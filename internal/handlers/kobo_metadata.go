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
		writeKoboJSON(r.Context(), w, http.StatusOK, []any{})
		return
	}
	tokenValue := auth.KoboTokenFromContext(r.Context())
	bookID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/library/"), "/metadata")
	if bookID == "" {
		writeKoboJSON(r.Context(), w, http.StatusNotFound, map[string]any{})
		return
	}

	book, err := h.DB.GetBook(r.Context(), bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeKoboJSON(r.Context(), w, http.StatusNotFound, map[string]any{})
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch book for kobo metadata", slog.Any(otelkeys.Error, err))
		writeKoboJSON(r.Context(), w, http.StatusInternalServerError, map[string]any{})
		return
	}

	rels, err := h.DB.LoadBookRelations(r.Context(), bookID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to load book relations for kobo metadata",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeKoboJSON(r.Context(), w, http.StatusInternalServerError, map[string]any{})
		return
	}

	base := schemeAndHost(r)
	downloadURLs := kobo.DownloadURLs(base, tokenValue, bookID, rels.Files)
	writeKoboJSON(r.Context(), w, http.StatusOK, []any{kobo.BookMetadata(book, rels.Authors, rels.Series, downloadURLs)})
}
