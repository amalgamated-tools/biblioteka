package handlers

import (
	"bytes"
	"context"
	"encoding/xml"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/opds"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// nowRFC3339 returns the current UTC time formatted as RFC 3339.
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// adaptNavEntities wraps a paginated list function so its results are converted
// to []opds.NavEntity for use with writeNamedEntityNavFeed. The toEntity
// callback maps each element of the source slice to an opds.NavEntity.
func adaptNavEntities[T any](
	listFn func(ctx context.Context, limit, offset int) ([]T, int, error),
	toEntity func(T) opds.NavEntity,
) func(ctx context.Context, limit, offset int) ([]opds.NavEntity, int, error) {
	return func(ctx context.Context, limit, offset int) ([]opds.NavEntity, int, error) {
		items, total, err := listFn(ctx, limit, offset)
		if err != nil {
			return nil, 0, err
		}
		entities := make([]opds.NavEntity, len(items))
		for i, item := range items {
			entities[i] = toEntity(item)
		}
		return entities, total, nil
	}
}

// writeOPDSError writes an error response for OPDS endpoints as a minimal Atom feed,
// so that OPDS clients always receive XML instead of JSON when an error occurs.
func writeOPDSError(r *http.Request, w http.ResponseWriter, status int, contentType, id, title string) {
	feed := &opds.Feed{
		XMLNS:     opds.XMLNSAtom,
		XMLNSOPDS: opds.XMLNSOPDS,
		ID:        id,
		Title:     title,
		Updated:   nowRFC3339(),
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)

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
	return requestScheme(r) + "://" + r.Host + "/opds"
}

func parsePage(r *http.Request) int {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func writeOPDSFeed(r *http.Request, w http.ResponseWriter, contentType string, feed *opds.Feed) {
	ctx := r.Context()
	var buf bytes.Buffer
	if _, err := buf.WriteString(xml.Header); err != nil {
		slog.ErrorContext(ctx, "failed to write OPDS XML header", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to generate feed")
		return
	}
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to encode feed", slog.Any(otelkeys.Error, err))
		writeOPDSError(r, w, http.StatusInternalServerError, contentType, "urn:biblioteka:opds:error", "failed to encode feed")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
