package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

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

func (h *OPDSHandler) allBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := opdsBaseURL(r)
	page := parsePage(r)
	offset := (page - 1) * opdsPageSize

	books, total, err := h.DB.ListBooksPaginated(ctx, opdsPageSize, offset)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list books", slog.Any(otelkeys.Error, err))
		writeOPDSError(r, w, http.StatusInternalServerError, opdsAcqContentType, baseURL+"/all", "Failed to list books")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        baseURL + "/all",
		Title:     "All Books",
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     paginationLinks(baseURL+"/all", page, total, opdsPageSize, opdsAcqContentType),
		Entries:   entries,
	}
	writeOPDSFeed(r, w, opdsAcqContentType, feed)
}

func (h *OPDSHandler) recentBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := opdsBaseURL(r)
	page := parsePage(r)
	offset := (page - 1) * opdsPageSize

	books, total, err := h.DB.ListRecentBooks(ctx, opdsPageSize, offset)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list recent books", slog.Any(otelkeys.Error, err))
		writeOPDSError(r, w, http.StatusInternalServerError, opdsAcqContentType, baseURL+"/recent", "Failed to list books")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        baseURL + "/recent",
		Title:     "Recent Books",
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     paginationLinks(baseURL+"/recent", page, total, opdsPageSize, opdsAcqContentType),
		Entries:   entries,
	}
	writeOPDSFeed(r, w, opdsAcqContentType, feed)
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
	baseURL := opdsBaseURL(r)
	page := parsePage(r)
	offset := (page - 1) * opdsPageSize

	author, err := h.DB.GetAuthor(ctx, authorID)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: author not found",
			slog.String(otelkeys.AuthorID, authorID),
			slog.Any(otelkeys.Error, err),
		)
		http.NotFound(w, r)
		return
	}

	books, total, err := h.DB.ListBooksByAuthorPaginated(ctx, authorID, opdsPageSize, offset)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list books by author", slog.Any(otelkeys.Error, err))
		writeOPDSError(r, w, http.StatusInternalServerError, opdsAcqContentType, baseURL+"/authors/"+authorID, "Failed to list books")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	selfURL := baseURL + "/authors/" + authorID
	links := paginationLinks(selfURL, page, total, opdsPageSize, opdsAcqContentType)
	links = append(links, opdsLink{Rel: relStart, Href: baseURL, Type: opdsNavContentType})
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        selfURL,
		Title:     "Books by " + author.Name,
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     links,
		Entries:   entries,
	}
	writeOPDSFeed(r, w, opdsAcqContentType, feed)
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
	baseURL := opdsBaseURL(r)
	page := parsePage(r)
	offset := (page - 1) * opdsPageSize

	series, err := h.DB.GetSeries(ctx, seriesID)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: series not found",
			slog.String(otelkeys.SeriesID, seriesID),
			slog.Any(otelkeys.Error, err),
		)
		http.NotFound(w, r)
		return
	}

	books, total, err := h.DB.ListBooksBySeriesPaginated(ctx, seriesID, opdsPageSize, offset)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list books by series", slog.Any(otelkeys.Error, err))
		writeOPDSError(r, w, http.StatusInternalServerError, opdsAcqContentType, baseURL+"/series/"+seriesID, "Failed to list books")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	selfURL := baseURL + "/series/" + seriesID
	links := paginationLinks(selfURL, page, total, opdsPageSize, opdsAcqContentType)
	links = append(links, opdsLink{Rel: relStart, Href: baseURL, Type: opdsNavContentType})
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        selfURL,
		Title:     series.Name,
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     links,
		Entries:   entries,
	}
	writeOPDSFeed(r, w, opdsAcqContentType, feed)
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
