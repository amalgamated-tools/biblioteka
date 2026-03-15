package handlers

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

const (
	xmlnsAtom       = "http://www.w3.org/2005/Atom"
	xmlnsOPDS       = "http://opds-spec.org/2010/catalog"
	xmlnsOpenSearch = "http://a9.com/-/spec/opensearch/1.1/"

	opdsNavContentType = "application/atom+xml;profile=opds-catalog;kind=navigation"
	opdsAcqContentType = "application/atom+xml;profile=opds-catalog;kind=acquisition"
	opdsSearchType     = "application/opensearchdescription+xml"

	opdsPageSize = 50

	relSelf        = "self"
	relStart       = "start"
	relNext        = "next"
	relPrevious    = "previous"
	relSearch      = "search"
	relSubsection  = "subsection"
	relAcquisition = "http://opds-spec.org/acquisition"
	relImage       = "http://opds-spec.org/image"
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

// OPDSHandler serves OPDS catalog feeds.
type OPDSHandler struct {
	DB *db.DB
}

// OPDS XML types

type opdsFeed struct {
	XMLName   xml.Name    `xml:"feed"`
	XMLNS     string      `xml:"xmlns,attr"`
	XMLNSOPDS string      `xml:"xmlns:opds,attr,omitempty"`
	ID        string      `xml:"id"`
	Title     string      `xml:"title"`
	Updated   string      `xml:"updated"`
	Author    *opdsAuthor `xml:"author,omitempty"`
	Links     []opdsLink  `xml:"link"`
	Entries   []opdsEntry `xml:"entry"`
}

type opdsEntry struct {
	Title   string       `xml:"title"`
	ID      string       `xml:"id"`
	Updated string       `xml:"updated"`
	Content *opdsContent `xml:"content,omitempty"`
	Authors []opdsAuthor `xml:"author,omitempty"`
	Links   []opdsLink   `xml:"link"`
}

type opdsLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr"`
}

type opdsAuthor struct {
	Name string `xml:"name"`
}

type opdsContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// HandleOPDS dispatches OPDS requests based on the URL path.
func (h *OPDSHandler) HandleOPDS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/opds")
	path = strings.TrimSuffix(path, "/")

	switch {
	case path == "" || path == "/":
		h.rootFeed(w, r)
	case path == "/all":
		h.allBooks(w, r)
	case path == "/recent":
		h.recentBooks(w, r)
	case path == "/authors":
		h.authorsFeed(w, r)
	case strings.HasPrefix(path, "/authors/"):
		h.authorBooks(w, r, strings.TrimPrefix(path, "/authors/"))
	case path == "/series":
		h.seriesFeed(w, r)
	case strings.HasPrefix(path, "/series/"):
		h.seriesBooks(w, r, strings.TrimPrefix(path, "/series/"))
	case path == "/search":
		if r.URL.Query().Get("q") != "" {
			h.searchResults(w, r)
		} else {
			h.openSearchDescription(w, r)
		}
	case strings.HasPrefix(path, "/download/"):
		h.downloadFile(w, r, strings.TrimPrefix(path, "/download/"))
	default:
		http.NotFound(w, r)
	}
}

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
		writeError(ctx, w, http.StatusInternalServerError, "failed to list books")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        baseURL + "/all",
		Title:     "All Books",
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     paginationLinks(baseURL+"/all", page, total, opdsPageSize),
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
		writeError(ctx, w, http.StatusInternalServerError, "failed to list books")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        baseURL + "/recent",
		Title:     "Recent Books",
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     paginationLinks(baseURL+"/recent", page, total, opdsPageSize),
		Entries:   entries,
	}
	writeOPDSFeed(r, w, opdsAcqContentType, feed)
}

func (h *OPDSHandler) authorsFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := opdsBaseURL(r)

	authors, err := h.DB.ListAuthors(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list authors", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to list authors")
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

	feed := &opdsFeed{
		XMLNS:   xmlnsAtom,
		ID:      baseURL + "/authors",
		Title:   "Authors",
		Updated: time.Now().UTC().Format(time.RFC3339),
		Links: []opdsLink{
			{Rel: relSelf, Href: baseURL + "/authors", Type: opdsNavContentType},
			{Rel: relStart, Href: baseURL, Type: opdsNavContentType},
		},
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
		writeError(ctx, w, http.StatusInternalServerError, "failed to list books")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	selfURL := baseURL + "/authors/" + authorID
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        selfURL,
		Title:     "Books by " + author.Name,
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     paginationLinks(selfURL, page, total, opdsPageSize),
		Entries:   entries,
	}
	writeOPDSFeed(r, w, opdsAcqContentType, feed)
}

func (h *OPDSHandler) seriesFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := opdsBaseURL(r)

	seriesList, err := h.DB.ListSeries(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list series", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to list series")
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

	feed := &opdsFeed{
		XMLNS:   xmlnsAtom,
		ID:      baseURL + "/series",
		Title:   "Series",
		Updated: time.Now().UTC().Format(time.RFC3339),
		Links: []opdsLink{
			{Rel: relSelf, Href: baseURL + "/series", Type: opdsNavContentType},
			{Rel: relStart, Href: baseURL, Type: opdsNavContentType},
		},
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
		writeError(ctx, w, http.StatusInternalServerError, "failed to list books")
		return
	}

	entries := h.bookEntries(ctx, books, baseURL)
	selfURL := baseURL + "/series/" + seriesID
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        selfURL,
		Title:     series.Name,
		Updated:   time.Now().UTC().Format(time.RFC3339),
		Links:     paginationLinks(selfURL, page, total, opdsPageSize),
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
		writeError(ctx, w, http.StatusInternalServerError, "search failed")
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
		Links:     paginationLinks(baseURL+"/search?q="+escapedQuery, page, total, opdsPageSize),
		Entries:   entries,
	}
	writeOPDSFeed(r, w, opdsAcqContentType, feed)
}

func (h *OPDSHandler) openSearchDescription(w http.ResponseWriter, r *http.Request) {
	baseURL := opdsBaseURL(r)

	type osURL struct {
		Type     string `xml:"type,attr"`
		Template string `xml:"template,attr"`
	}
	type osDesc struct {
		XMLName     xml.Name `xml:"OpenSearchDescription"`
		XMLNS       string   `xml:"xmlns,attr"`
		ShortName   string   `xml:"ShortName"`
		Description string   `xml:"Description"`
		URL         osURL    `xml:"Url"`
	}

	desc := osDesc{
		XMLNS:       xmlnsOpenSearch,
		ShortName:   "Biblioteka",
		Description: "Search books in Biblioteka",
		URL: osURL{
			Type:     opdsAcqContentType,
			Template: baseURL + "/search?q={searchTerms}",
		},
	}

	w.Header().Set("Content-Type", opdsSearchType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(desc)
}

func (h *OPDSHandler) downloadFile(w http.ResponseWriter, r *http.Request, fileID string) {
	ctx := r.Context()

	bf, err := h.DB.GetBookFile(ctx, fileID)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: book file not found",
			slog.String(otelkeys.BookFileID, fileID),
			slog.Any(otelkeys.Error, err),
		)
		http.NotFound(w, r)
		return
	}

	mimeType := fileTypeMIME[strings.ToLower(bf.FileType)]
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	f, err := os.Open(bf.FilePath)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to open book file",
			slog.String(otelkeys.BookFileID, fileID),
			slog.Any(otelkeys.Error, err),
		)
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to stat book file",
			slog.String(otelkeys.BookFileID, fileID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to read file")
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": bf.FileName}))
	http.ServeContent(w, r, bf.FileName, stat.ModTime(), f)
}

// bookEntries converts a slice of books into OPDS entry elements, including
// authors and download links for each book. Uses batch queries to avoid N+1.
func (h *OPDSHandler) bookEntries(ctx context.Context, books []db.Book, baseURL string) []opdsEntry {
	if len(books) == 0 {
		return nil
	}

	// Collect book IDs for batch loading.
	bookIDs := make([]string, len(books))
	for i, b := range books {
		bookIDs[i] = b.ID
	}

	// Batch-load authors and files in two queries.
	authorsByBook, err := h.DB.GetAuthorsForBooks(ctx, bookIDs)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to batch-load book authors",
			slog.Any(otelkeys.Error, err),
		)
		authorsByBook = nil
	}

	filesByBook, err := h.DB.GetFilesForBooks(ctx, bookIDs)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to batch-load book files",
			slog.Any(otelkeys.Error, err),
		)
		filesByBook = nil
	}

	entries := make([]opdsEntry, 0, len(books))
	for _, book := range books {
		entry := opdsEntry{
			Title:   book.Title,
			ID:      baseURL + "/books/" + book.ID,
			Updated: book.UpdatedAt.Format(time.RFC3339),
		}

		if book.Description != nil && *book.Description != "" {
			entry.Content = &opdsContent{Type: "text", Value: *book.Description}
		}

		// Add authors from batch result.
		if authorsByBook != nil {
			for _, a := range authorsByBook[book.ID] {
				entry.Authors = append(entry.Authors, opdsAuthor{Name: a.Name})
			}
		}

		// Add cover image link
		if book.CoverImageURL != nil && *book.CoverImageURL != "" {
			coverType := coverMIMEType(*book.CoverImageURL)
			entry.Links = append(entry.Links, opdsLink{
				Rel:  relImage,
				Href: *book.CoverImageURL,
				Type: coverType,
			})
		}

		// Add download links from batch result.
		if filesByBook != nil {
			for _, f := range filesByBook[book.ID] {
				mimeType := fileTypeMIME[strings.ToLower(f.FileType)]
				if mimeType == "" {
					mimeType = "application/octet-stream"
				}
				entry.Links = append(entry.Links, opdsLink{
					Rel:  relAcquisition,
					Href: baseURL + "/download/" + f.ID,
					Type: mimeType,
				})
			}
		}

		entries = append(entries, entry)
	}
	return entries
}

// Helper functions

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

func paginationLinks(selfURL string, page, total, pageSize int) []opdsLink {
	// For URLs that already have query params (like search), use & instead of ?
	sep := "?"
	if strings.Contains(selfURL, "?") {
		sep = "&"
	}

	links := []opdsLink{
		{Rel: relSelf, Href: selfURL + sep + "page=" + strconv.Itoa(page), Type: opdsAcqContentType},
	}

	if page > 1 {
		links = append(links, opdsLink{
			Rel:  relPrevious,
			Href: selfURL + sep + "page=" + strconv.Itoa(page-1),
			Type: opdsAcqContentType,
		})
	}

	totalPages := (total + pageSize - 1) / pageSize
	if page < totalPages {
		links = append(links, opdsLink{
			Rel:  relNext,
			Href: selfURL + sep + "page=" + strconv.Itoa(page+1),
			Type: opdsAcqContentType,
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
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to encode feed")
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
