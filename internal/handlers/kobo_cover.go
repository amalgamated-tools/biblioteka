package handlers

import (
	"bytes"
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

	contentType, data, err := decodeDataURL(*book.CoverImageURL)
	if err == nil {
		effectiveContentType := contentType
		declaredIsImage := strings.HasPrefix(contentType, "image/")
		if len(data) > 0 {
			sniffed := http.DetectContentType(data)
			sniffedIsImage := strings.HasPrefix(sniffed, "image/")
			sniffedIsXML := strings.HasPrefix(sniffed, "text/xml") || strings.HasPrefix(sniffed, "application/xml")

			if declaredIsImage {
				if sniffedIsImage {
					// Prefer a more specific sniffed image type when available.
					effectiveContentType = sniffed
				} else if strings.EqualFold(contentType, "image/svg+xml") && sniffedIsXML {
					// Allow SVG declared as image/svg+xml even when sniffed as XML.
					effectiveContentType = contentType
				} else {
					slog.WarnContext(r.Context(), "cover data does not look like an image",
						slog.String(otelkeys.BookID, bookID),
						slog.String(otelkeys.ContentType, sniffed),
					)
					http.Error(w, "invalid cover image", http.StatusInternalServerError)
					return
				}
			} else {
				if sniffedIsImage {
					// Declared non-image, but sniffed as image: trust the sniffed image type.
					effectiveContentType = sniffed
				} else {
					slog.WarnContext(r.Context(), "cover data does not look like an image",
						slog.String(otelkeys.BookID, bookID),
						slog.String(otelkeys.ContentType, sniffed),
					)
					http.Error(w, "invalid cover image", http.StatusInternalServerError)
					return
				}
			}
		}
		if !strings.HasPrefix(effectiveContentType, "image/") {
			slog.WarnContext(r.Context(), "non-image content type in data URL for cover image",
				slog.String(otelkeys.BookID, bookID),
				slog.String(otelkeys.ContentType, contentType),
			)
			http.Error(w, "invalid cover image", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", effectiveContentType)
		http.ServeContent(w, r, "cover", book.UpdatedAt.Time, bytes.NewReader(data))
		return
	}
	if !errors.Is(err, errNotDataURL) {
		slog.WarnContext(r.Context(), "malformed data URL for cover image",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		http.Error(w, "invalid cover image", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, *book.CoverImageURL, http.StatusTemporaryRedirect)
}
