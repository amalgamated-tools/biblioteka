package handlers

import (
	"bytes"
	"encoding/xml"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// MIME types for common ebook formats.
var fileTypeMIME = map[string]string{
	"epub": "application/epub+zip",
	"pdf":  "application/pdf",
	"mobi": "application/x-mobipocket-ebook",
	"azw3": "application/vnd.amazon.ebook",
	"cbz":  "application/vnd.comicbook+zip",
	"cbr":  "application/vnd.comicbook-rar",
	"fb2":  "application/x-fictionbook+xml",
	"txt":  "text/plain",
	"djvu": "image/vnd.djvu",
}

// writeOPDSError writes an error response for OPDS endpoints as a minimal Atom feed,
// so that OPDS clients always receive XML instead of JSON when an error occurs.
func writeOPDSError(r *http.Request, w http.ResponseWriter, status int, contentType, id, title string) {
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        id,
		Title:     title,
		Updated:   time.Now().UTC().Format(time.RFC3339),
	}

	var buf bytes.Buffer
	if _, err := buf.WriteString(xml.Header); err != nil {
		slog.ErrorContext(r.Context(), "failed to write OPDS XML header",
			slog.Any(otelkeys.Error, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode OPDS error feed",
			slog.Any(otelkeys.Error, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)

	if _, err := w.Write(buf.Bytes()); err != nil {
		slog.ErrorContext(r.Context(), "failed to write OPDS error response body",
			slog.Any(otelkeys.Error, err))
	}
}

func opdsBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "http" || proto == "https" {
		scheme = proto
	}
	return scheme + "://" + r.Host + "/opds"
}

func parsePage(r *http.Request) int {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func paginationLinks(selfURL string, page, total, pageSize int, contentType string) []opdsLink {
	// For URLs that already have query params (like search), use & instead of ?
	sep := "?"
	if strings.Contains(selfURL, "?") {
		sep = "&"
	}

	links := []opdsLink{
		{Rel: relSelf, Href: selfURL + sep + "page=" + strconv.Itoa(page), Type: contentType},
	}

	if page > 1 {
		links = append(links, opdsLink{
			Rel:  relPrevious,
			Href: selfURL + sep + "page=" + strconv.Itoa(page-1),
			Type: contentType,
		})
	}

	totalPages := (total + pageSize - 1) / pageSize
	if page < totalPages {
		links = append(links, opdsLink{
			Rel:  relNext,
			Href: selfURL + sep + "page=" + strconv.Itoa(page+1),
			Type: contentType,
		})
	}

	return links
}

func writeOPDSFeed(r *http.Request, w http.ResponseWriter, contentType string, feed *opdsFeed) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		slog.ErrorContext(r.Context(), "OPDS: failed to encode feed", slog.Any(otelkeys.Error, err))
		writeOPDSError(r, w, http.StatusInternalServerError, contentType, "urn:biblioteka:opds:error", "failed to encode feed")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// coverMIMEType returns the MIME type for a cover image URL based on its extension.
func coverMIMEType(imageURL string) string {
	switch strings.ToLower(path.Ext(imageURL)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	default:
		return "image/jpeg"
	}
}
