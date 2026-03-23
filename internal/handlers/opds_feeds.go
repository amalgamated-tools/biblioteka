package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

func (h *OPDSHandler) rootFeed(w http.ResponseWriter, r *http.Request) {
	baseURL := opdsBaseURL(r)
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        baseURL + "/",
		Title:     "Biblioteka OPDS Catalog",
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links: []opdsLink{
			{Rel: relSelf, Href: baseURL, Type: opdsNavContentType},
			{Rel: relStart, Href: baseURL, Type: opdsNavContentType},
			{Rel: relSearch, Href: baseURL + "/search", Type: opdsSearchType},
		},
		Entries: []opdsEntry{
			{
				Title:   "All Books",
				ID:      baseURL + "/all",
				Updated: time.Now().UTC().Format(time.RFC3339),
				Content: &opdsContent{Type: "text", Value: "Browse all books"},
				Links:   []opdsLink{{Rel: relSubsection, Href: baseURL + "/all", Type: opdsAcqContentType}},
			},
			{
				Title:   "Recent Books",
				ID:      baseURL + "/recent",
				Updated: time.Now().UTC().Format(time.RFC3339),
				Content: &opdsContent{Type: "text", Value: "Recently added books"},
				Links:   []opdsLink{{Rel: relSubsection, Href: baseURL + "/recent", Type: opdsAcqContentType}},
			},
			{
				Title:   "Authors",
				ID:      baseURL + "/authors",
				Updated: time.Now().UTC().Format(time.RFC3339),
				Content: &opdsContent{Type: "text", Value: "Browse by author"},
				Links:   []opdsLink{{Rel: relSubsection, Href: baseURL + "/authors", Type: opdsNavContentType}},
			},
			{
				Title:   "Series",
				ID:      baseURL + "/series",
				Updated: time.Now().UTC().Format(time.RFC3339),
				Content: &opdsContent{Type: "text", Value: "Browse by series"},
				Links:   []opdsLink{{Rel: relSubsection, Href: baseURL + "/series", Type: opdsNavContentType}},
			},
		},
	}
	writeOPDSFeed(r, w, opdsNavContentType, feed)
}

// writeBooksFeed is a shared helper for book-listing OPDS acquisition feeds.
// selfPath is the path after /opds (e.g. "/all" or "/authors/<id>").
// extraLinks are appended after the pagination links (e.g. a relStart link back to root).
// listFn must return (books, totalCount, error) for the given limit and offset.
func (h *OPDSHandler) writeBooksFeed(
	w http.ResponseWriter, r *http.Request,
	selfPath, title string,
	extraLinks []opdsLink,
	listFn func(ctx context.Context, limit, offset int) ([]db.Book, int, error),
) {
	ctx := r.Context()
	baseURL := opdsBaseURL(r)
	page := parsePage(r)
	offset := (page - 1) * opdsPageSize
	selfURL := baseURL + selfPath

	books, total, err := listFn(ctx, opdsPageSize, offset)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"OPDS: failed to list books",
			slog.String(otelkeys.URL, selfURL),
			slog.String(otelkeys.Title, title),
			slog.Any(otelkeys.Error, err),
		)
		writeOPDSError(r, w, http.StatusInternalServerError, opdsAcqContentType, selfURL, "Failed to list books")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	links := paginationLinks(selfURL, page, total, opdsPageSize, opdsAcqContentType)
	links = append(links, extraLinks...)
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        selfURL,
		Title:     title,
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     links,
		Entries:   entries,
	}
	writeOPDSFeed(r, w, opdsAcqContentType, feed)
}

func (h *OPDSHandler) allBooks(w http.ResponseWriter, r *http.Request) {
	h.writeBooksFeed(w, r, "/all", "All Books", nil, h.DB.ListBooksPaginated)
}

func (h *OPDSHandler) recentBooks(w http.ResponseWriter, r *http.Request) {
	h.writeBooksFeed(w, r, "/recent", "Recent Books", nil, h.DB.ListRecentBooks)
}

func (h *OPDSHandler) authorsFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := opdsBaseURL(r)
	page := parsePage(r)
	offset := (page - 1) * opdsPageSize

	authors, total, err := h.DB.ListAuthorsPaginated(ctx, opdsPageSize, offset)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list authors", slog.Any(otelkeys.Error, err))
		writeOPDSError(r, w, http.StatusInternalServerError, opdsNavContentType, baseURL+"/authors", "failed to list authors")
		return
	}

	entries := make([]opdsEntry, 0, len(authors))
	for _, a := range authors {
		entries = append(entries, opdsEntry{
			Title:   a.Name,
			ID:      baseURL + "/authors/" + a.ID,
			Updated: a.UpdatedAt.Format(time.RFC3339),
			Links: []opdsLink{
				{Rel: relSubsection, Href: baseURL + "/authors/" + a.ID, Type: opdsAcqContentType},
			},
		})
	}

	selfURL := baseURL + "/authors"
	links := paginationLinks(selfURL, page, total, opdsPageSize, opdsNavContentType)
	links = append(links, opdsLink{Rel: relStart, Href: baseURL, Type: opdsNavContentType})

	feed := &opdsFeed{
		XMLNS:   xmlnsAtom,
		ID:      selfURL,
		Title:   "Authors",
		Updated: time.Now().UTC().Format(time.RFC3339),
		Links:   links,
		Entries: entries,
	}
	writeOPDSFeed(r, w, opdsNavContentType, feed)
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

	baseURL := opdsBaseURL(r)
	extraLinks := []opdsLink{{Rel: relStart, Href: baseURL, Type: opdsNavContentType}}
	h.writeBooksFeed(w, r, "/authors/"+authorID, "Books by "+author.Name, extraLinks,
		func(c context.Context, limit, offset int) ([]db.Book, int, error) {
			return h.DB.ListBooksByAuthorPaginated(c, authorID, limit, offset)
		},
	)
}

func (h *OPDSHandler) seriesFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := opdsBaseURL(r)
	page := parsePage(r)
	offset := (page - 1) * opdsPageSize

	seriesList, total, err := h.DB.ListSeriesPaginated(ctx, opdsPageSize, offset)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list series", slog.Any(otelkeys.Error, err))
		writeOPDSError(r, w, http.StatusInternalServerError, opdsNavContentType, baseURL+"/series", "Failed to list series")
		return
	}

	entries := make([]opdsEntry, 0, len(seriesList))
	for _, s := range seriesList {
		entries = append(entries, opdsEntry{
			Title:   s.Name,
			ID:      baseURL + "/series/" + s.ID,
			Updated: s.UpdatedAt.Format(time.RFC3339),
			Links: []opdsLink{
				{Rel: relSubsection, Href: baseURL + "/series/" + s.ID, Type: opdsAcqContentType},
			},
		})
	}

	selfURL := baseURL + "/series"
	links := paginationLinks(selfURL, page, total, opdsPageSize, opdsNavContentType)
	links = append(links, opdsLink{Rel: relStart, Href: baseURL, Type: opdsNavContentType})

	feed := &opdsFeed{
		XMLNS:   xmlnsAtom,
		ID:      selfURL,
		Title:   "Series",
		Updated: time.Now().UTC().Format(time.RFC3339),
		Links:   links,
		Entries: entries,
	}
	writeOPDSFeed(r, w, opdsNavContentType, feed)
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

	baseURL := opdsBaseURL(r)
	extraLinks := []opdsLink{{Rel: relStart, Href: baseURL, Type: opdsNavContentType}}
	h.writeBooksFeed(w, r, "/series/"+seriesID, series.Name, extraLinks,
		func(c context.Context, limit, offset int) ([]db.Book, int, error) {
			return h.DB.ListBooksBySeriesPaginated(c, seriesID, limit, offset)
		},
	)
}

func (h *OPDSHandler) searchResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := opdsBaseURL(r)
	query := r.URL.Query().Get("q")
	page := parsePage(r)
	offset := (page - 1) * opdsPageSize

	books, total, err := h.DB.SearchBooks(ctx, query, opdsPageSize, offset)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: search failed", slog.Any(otelkeys.Error, err))
		writeOPDSError(r, w, http.StatusInternalServerError, opdsAcqContentType, baseURL+"/search", "Search failed")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	escapedQuery := url.QueryEscape(query)
	selfURL := baseURL + "/search?q=" + escapedQuery
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        selfURL,
		Title:     fmt.Sprintf("Search: %s", query),
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     paginationLinks(baseURL+"/search?q="+escapedQuery, page, total, opdsPageSize, opdsAcqContentType),
		Entries:   entries,
	}
	writeOPDSFeed(r, w, opdsAcqContentType, feed)
}
