package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	opdspkg "github.com/amalgamated-tools/biblioteka/internal/opds"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

func (h *OPDSHandler) rootFeed(w http.ResponseWriter, r *http.Request) {
	baseURL := opdsBaseURL(r)
	feed := &opdspkg.Feed{
		XMLNS:     opdspkg.XMLNSAtom,
		XMLNSOPDS: opdspkg.XMLNSOPDS,
		ID:        baseURL + "/",
		Title:     "Biblioteka OPDS Catalog",
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links: []opdspkg.Link{
			{Rel: opdspkg.RelSelf, Href: baseURL, Type: opdspkg.NavContentType},
			{Rel: opdspkg.RelStart, Href: baseURL, Type: opdspkg.NavContentType},
			{Rel: opdspkg.RelSearch, Href: baseURL + "/search", Type: opdspkg.SearchType},
		},
		Entries: []opdspkg.Entry{
			{
				Title:   "All Books",
				ID:      baseURL + "/all",
				Updated: time.Now().UTC().Format(time.RFC3339),
				Content: &opdspkg.Content{Type: "text", Value: "Browse all books"},
				Links:   []opdspkg.Link{{Rel: opdspkg.RelSubsection, Href: baseURL + "/all", Type: opdspkg.AcqContentType}},
			},
			{
				Title:   "Recent Books",
				ID:      baseURL + "/recent",
				Updated: time.Now().UTC().Format(time.RFC3339),
				Content: &opdspkg.Content{Type: "text", Value: "Recently added books"},
				Links:   []opdspkg.Link{{Rel: opdspkg.RelSubsection, Href: baseURL + "/recent", Type: opdspkg.AcqContentType}},
			},
			{
				Title:   "Authors",
				ID:      baseURL + "/authors",
				Updated: time.Now().UTC().Format(time.RFC3339),
				Content: &opdspkg.Content{Type: "text", Value: "Browse by author"},
				Links:   []opdspkg.Link{{Rel: opdspkg.RelSubsection, Href: baseURL + "/authors", Type: opdspkg.NavContentType}},
			},
			{
				Title:   "Series",
				ID:      baseURL + "/series",
				Updated: time.Now().UTC().Format(time.RFC3339),
				Content: &opdspkg.Content{Type: "text", Value: "Browse by series"},
				Links:   []opdspkg.Link{{Rel: opdspkg.RelSubsection, Href: baseURL + "/series", Type: opdspkg.NavContentType}},
			},
		},
	}
	writeOPDSFeed(r, w, opdspkg.NavContentType, feed)
}

// writeBooksFeed is a shared helper for book-listing OPDS acquisition feeds.
// selfPath is the path after /opds (e.g. "/all" or "/authors/<id>").
// extraLinks are appended after the pagination links (e.g. an opdspkg.RelStart link back to root).
// listFn must return (books, totalCount, error) for the given limit and offset.
func (h *OPDSHandler) writeBooksFeed(
	w http.ResponseWriter, r *http.Request,
	selfPath, title string,
	extraLinks []opdspkg.Link,
	listFn func(ctx context.Context, limit, offset int) ([]db.Book, int, error),
) {
	ctx := r.Context()
	baseURL := opdsBaseURL(r)
	page := parsePage(r)
	offset := (page - 1) * opdspkg.PageSize
	selfURL := baseURL + selfPath

	books, total, err := listFn(ctx, opdspkg.PageSize, offset)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"OPDS: failed to list books",
			slog.String(otelkeys.URL, selfURL),
			slog.String(otelkeys.Title, title),
			slog.Any(otelkeys.Error, err),
		)
		writeOPDSError(r, w, http.StatusInternalServerError, opdspkg.AcqContentType, selfURL, "Failed to list books")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	links := opdspkg.PaginationLinks(selfURL, page, total, opdspkg.PageSize, opdspkg.AcqContentType)
	links = append(links, extraLinks...)
	feed := &opdspkg.Feed{
		XMLNS:     opdspkg.XMLNSAtom,
		XMLNSOPDS: opdspkg.XMLNSOPDS,
		ID:        selfURL,
		Title:     title,
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     links,
		Entries:   entries,
	}
	writeOPDSFeed(r, w, opdspkg.AcqContentType, feed)
}

func (h *OPDSHandler) allBooks(w http.ResponseWriter, r *http.Request) {
	h.writeBooksFeed(w, r, "/all", "All Books", nil, h.DB.ListBooksPaginated)
}

func (h *OPDSHandler) recentBooks(w http.ResponseWriter, r *http.Request) {
	h.writeBooksFeed(w, r, "/recent", "Recent Books", nil, h.DB.ListRecentBooks)
}

// writeNamedEntityNavFeed is a shared helper for paginated OPDS navigation feeds
// that list named entities (authors, series, etc.).
// pathSegment is the path component after /opds (e.g. "authors" or "series").
// listFn must return (entities, totalCount, error) for the given limit and offset.
func (h *OPDSHandler) writeNamedEntityNavFeed(
	w http.ResponseWriter, r *http.Request,
	pathSegment, title string,
	listFn func(ctx context.Context, limit, offset int) ([]opdspkg.NavEntity, int, error),
) {
	ctx := r.Context()
	baseURL := opdsBaseURL(r)
	page := parsePage(r)
	offset := (page - 1) * opdspkg.PageSize
	selfURL := baseURL + "/" + pathSegment

	entities, total, err := listFn(ctx, opdspkg.PageSize, offset)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list entities",
			slog.String(otelkeys.EntityType, pathSegment),
			slog.String(otelkeys.URL, selfURL),
			slog.String(otelkeys.Title, title),
			slog.Any(otelkeys.Error, err),
		)
		writeOPDSError(r, w, http.StatusInternalServerError, opdspkg.NavContentType, selfURL, fmt.Sprintf("Failed to list %s", pathSegment))
		return
	}
	entries := make([]opdspkg.Entry, 0, len(entities))
	for _, e := range entities {
		entryURL := selfURL + "/" + e.ID
		entries = append(entries, opdspkg.Entry{
			Title:   e.Name,
			ID:      entryURL,
			Updated: e.Updated,
			Links:   []opdspkg.Link{{Rel: opdspkg.RelSubsection, Href: entryURL, Type: opdspkg.AcqContentType}},
		})
	}

	links := opdspkg.PaginationLinks(selfURL, page, total, opdspkg.PageSize, opdspkg.NavContentType)
	links = append(links, opdspkg.Link{Rel: opdspkg.RelStart, Href: baseURL, Type: opdspkg.NavContentType})

	feed := &opdspkg.Feed{
		XMLNS:   opdspkg.XMLNSAtom,
		ID:      selfURL,
		Title:   title,
		Updated: time.Now().UTC().Format(time.RFC3339),
		Links:   links,
		Entries: entries,
	}
	writeOPDSFeed(r, w, opdspkg.NavContentType, feed)
}

func (h *OPDSHandler) authorsFeed(w http.ResponseWriter, r *http.Request) {
	h.writeNamedEntityNavFeed(w, r, "authors", "Authors",
		func(ctx context.Context, limit, offset int) ([]opdspkg.NavEntity, int, error) {
			authors, total, err := h.DB.ListAuthorsPaginated(ctx, limit, offset)
			if err != nil {
				return nil, 0, err
			}
			entities := make([]opdspkg.NavEntity, len(authors))
			for i, a := range authors {
				entities[i] = opdspkg.NavEntity{ID: a.ID, Name: a.Name, Updated: a.UpdatedAt.Format(time.RFC3339)}
			}
			return entities, total, nil
		},
	)
}

func (h *OPDSHandler) authorBooks(w http.ResponseWriter, r *http.Request, authorID string) {
	ctx := r.Context()
	author, err := h.DB.GetAuthor(ctx, authorID)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: author not found",
			slog.String(otelkeys.AuthorID, authorID),
			slog.Any(otelkeys.Error, err),
		)
		http.NotFound(w, r)
		return
	}

	h.writeBooksForEntity(w, r, "/authors/"+authorID, "Books by "+author.Name,
		func(c context.Context, limit, offset int) ([]db.Book, int, error) {
			return h.DB.ListBooksByAuthorPaginated(c, authorID, limit, offset)
		},
	)
}

func (h *OPDSHandler) seriesFeed(w http.ResponseWriter, r *http.Request) {
	h.writeNamedEntityNavFeed(w, r, "series", "Series",
		func(ctx context.Context, limit, offset int) ([]opdspkg.NavEntity, int, error) {
			seriesList, total, err := h.DB.ListSeriesPaginated(ctx, limit, offset)
			if err != nil {
				return nil, 0, err
			}
			entities := make([]opdspkg.NavEntity, len(seriesList))
			for i, s := range seriesList {
				entities[i] = opdspkg.NavEntity{ID: s.ID, Name: s.Name, Updated: s.UpdatedAt.Format(time.RFC3339)}
			}
			return entities, total, nil
		},
	)
}

func (h *OPDSHandler) seriesBooks(w http.ResponseWriter, r *http.Request, seriesID string) {
	ctx := r.Context()
	series, err := h.DB.GetSeries(ctx, seriesID)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: series not found",
			slog.String(otelkeys.SeriesID, seriesID),
			slog.Any(otelkeys.Error, err),
		)
		http.NotFound(w, r)
		return
	}

	h.writeBooksForEntity(w, r, "/series/"+seriesID, series.Name,
		func(c context.Context, limit, offset int) ([]db.Book, int, error) {
			return h.DB.ListBooksBySeriesPaginated(c, seriesID, limit, offset)
		},
	)
}

// writeBooksForEntity is a shared helper for entity-scoped OPDS book feeds
// (e.g. books by a given author, or books in a given series). It appends an
// opdspkg.RelStart link back to the OPDS root and delegates to writeBooksFeed.
func (h *OPDSHandler) writeBooksForEntity(
	w http.ResponseWriter, r *http.Request,
	pathSegment, title string,
	listFn func(ctx context.Context, limit, offset int) ([]db.Book, int, error),
) {
	baseURL := opdsBaseURL(r)
	extraLinks := []opdspkg.Link{{Rel: opdspkg.RelStart, Href: baseURL, Type: opdspkg.NavContentType}}
	h.writeBooksFeed(w, r, pathSegment, title, extraLinks, listFn)
}

func (h *OPDSHandler) searchResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := opdsBaseURL(r)
	query := r.URL.Query().Get("q")
	page := parsePage(r)
	offset := (page - 1) * opdspkg.PageSize

	books, total, err := h.DB.SearchBooks(ctx, query, opdspkg.PageSize, offset)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: search failed", slog.Any(otelkeys.Error, err))
		writeOPDSError(r, w, http.StatusInternalServerError, opdspkg.AcqContentType, baseURL+"/search", "Search failed")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	escapedQuery := url.QueryEscape(query)
	selfURL := baseURL + "/search?q=" + escapedQuery
	feed := &opdspkg.Feed{
		XMLNS:     opdspkg.XMLNSAtom,
		XMLNSOPDS: opdspkg.XMLNSOPDS,
		ID:        selfURL,
		Title:     fmt.Sprintf("Search: %s", query),
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     opdspkg.PaginationLinks(baseURL+"/search?q="+escapedQuery, page, total, opdspkg.PageSize, opdspkg.AcqContentType),
		Entries:   entries,
	}
	writeOPDSFeed(r, w, opdspkg.AcqContentType, feed)
}
