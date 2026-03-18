package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// HandleCoverImage handles requests for book cover images.
// Path: /covers/{bookID}/{width}/{height}/{quality}/{isGreyscale}/image.jpg
// If the book has a cover_image_url, it redirects there; otherwise returns 404.
func (h *KoboHandler) HandleCoverImage(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/covers/")
	bookID := strings.SplitN(trimmed, "/", 2)[0]
	if bookID == "" {
		http.NotFound(w, r)
		return
	}

	book, err := h.DB.GetBook(r.Context(), bookID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(r.Context(), "failed to fetch book for kobo cover image",
				slog.String(otelkeys.BookID, bookID),
				slog.Any(otelkeys.Error, err),
			)
		}
		http.NotFound(w, r)
		return
	}
	if book.CoverImageURL == nil || *book.CoverImageURL == "" {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, *book.CoverImageURL, http.StatusTemporaryRedirect)
}
