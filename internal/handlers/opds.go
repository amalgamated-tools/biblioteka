package handlers

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/crypto/bcrypt"
)

// OPDS namespace constants.
const (
	opdsRelStart  = "start"
	opdsRelSelf   = "self"
	opdsRelUp     = "up"
	opdsRelSearch = "search"

	opdsNavigationType  = "application/atom+xml;profile=opds-catalog;kind=navigation"
	opdsAcquisitionType = "application/atom+xml;profile=opds-catalog;kind=acquisition"
	opdsRelSubsection   = "subsection"
	opdsRelAcquisition  = "http://opds-spec.org/acquisition"
	opdsRelImage        = "http://opds-spec.org/image"
	opdsRelThumbnail    = "http://opds-spec.org/image/thumbnail"
)

// Basic Auth rate limiting for OPDS middleware.
//
// This is a simple per-client-IP sliding window limiter used to reduce
// brute-force attempts against the OPDS Basic Auth endpoint. JWT-based
// authentication bypasses this limiter.
const (
	opdsAuthMaxAttempts = 10
	opdsAuthWindow      = time.Minute
)

type opdsAuthRateState struct {
	count       int
	windowStart time.Time
}

var (
	opdsAuthRateMu    sync.Mutex
	opdsAuthRateStateMap = make(map[string]*opdsAuthRateState)
)

func opdsAuthClientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// allowOPDSBasicAuth applies a simple per-IP rate limit for Basic Auth attempts.
func allowOPDSBasicAuth(r *http.Request) bool {
	key := opdsAuthClientKey(r)
	now := time.Now()

	opdsAuthRateMu.Lock()
	defer opdsAuthRateMu.Unlock()

	state, ok := opdsAuthRateStateMap[key]
	if !ok || now.Sub(state.windowStart) > opdsAuthWindow {
		opdsAuthRateStateMap[key] = &opdsAuthRateState{
			count:       1,
			windowStart: now,
		}
		return true
	}

	if state.count >= opdsAuthMaxAttempts {
		return false
	}

	state.count++
	return true
}

// opdsFeed represents an OPDS/Atom feed document.
type opdsFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	ID      string      `xml:"id"`
	Title   string      `xml:"title"`
	Updated string      `xml:"updated"`
	Author  *opdsAuthor `xml:"author,omitempty"`
	Links   []opdsLink  `xml:"link"`
	Entries []opdsEntry `xml:"entry"`
}

// opdsEntry represents a single entry in an OPDS feed.
type opdsEntry struct {
	Title      string       `xml:"title"`
	ID         string       `xml:"id"`
	Updated    string       `xml:"updated"`
	Content    *opdsContent `xml:"content,omitempty"`
	Summary    string       `xml:"summary,omitempty"`
	Authors    []opdsAuthor `xml:"author,omitempty"`
	Links      []opdsLink   `xml:"link"`
	Publisher  string       `xml:"http://purl.org/dc/terms/ publisher,omitempty"`
	Language   string       `xml:"http://purl.org/dc/terms/ language,omitempty"`
	Issued     string       `xml:"http://purl.org/dc/terms/ issued,omitempty"`
	Identifier string       `xml:"http://purl.org/dc/terms/ identifier,omitempty"`
}

// opdsContent represents the content element of an entry.
type opdsContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// opdsAuthor represents an author element in an Atom feed.
type opdsAuthor struct {
	Name string `xml:"name"`
	URI  string `xml:"uri,omitempty"`
}

// opdsLink represents a link element in an Atom feed.
type opdsLink struct {
	Rel   string `xml:"rel,attr,omitempty"`
	Href  string `xml:"href,attr"`
	Type  string `xml:"type,attr,omitempty"`
	Title string `xml:"title,attr,omitempty"`
}

// OPDSHandler holds dependencies for OPDS endpoints.
type OPDSHandler struct {
	DB  *db.DB
	JWT *auth.JWTManager
}

// Middleware authenticates OPDS requests via JWT Bearer token, the
// biblioteka_token cookie, or HTTP Basic Auth (email:password).
func (h *OPDSHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Try JWT first (Bearer header or cookie).
		token, _ := auth.ExtractToken(r)
		if token != "" {
			claims, err := h.JWT.ValidateToken(ctx, token)
			if err == nil {
				slog.DebugContext(ctx, "OPDS: JWT authentication successful", slog.String(otelkeys.UserID, claims.UserID))
				ctx = auth.ContextWithUserID(ctx, claims.UserID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			slog.DebugContext(ctx, "OPDS: invalid JWT token", slog.Any(otelkeys.Error, err))
		}

		// Fall back to HTTP Basic Auth with rate limiting to prevent brute-force attacks.
		if !allowOPDSBasicAuth(r) {
			slog.WarnContext(ctx, "OPDS: Basic Auth rate limit exceeded", slog.String("remote_addr", r.RemoteAddr))
			w.Header().Set("WWW-Authenticate", `Basic realm="Biblioteka"`)
			http.Error(w, "too many authentication attempts, please try again later", http.StatusTooManyRequests)
			return
		}

		email, password, ok := r.BasicAuth()
		if !ok || email == "" || password == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Biblioteka"`)
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}

		user, err := h.DB.GetUserByEmail(ctx, email)
		if err != nil {
			if err == sql.ErrNoRows {
				slog.InfoContext(ctx, "OPDS: user not found", slog.String(otelkeys.Email, email))
			} else {
				slog.ErrorContext(ctx, "OPDS: failed to look up user", slog.Any(otelkeys.Error, err))
			}
			w.Header().Set("WWW-Authenticate", `Basic realm="Biblioteka"`)
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			slog.InfoContext(ctx, "OPDS: invalid password", slog.String(otelkeys.Email, email))
			w.Header().Set("WWW-Authenticate", `Basic realm="Biblioteka"`)
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		slog.DebugContext(ctx, "OPDS: Basic Auth successful", slog.String(otelkeys.UserID, user.ID))
		ctx = auth.ContextWithUserID(ctx, user.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// writeOPDS writes an OPDS XML feed response.
func writeOPDS(ctx context.Context, w http.ResponseWriter, feed *opdsFeed) {
	w.Header().Set("Content-Type", "application/atom+xml;charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to encode feed", slog.Any(otelkeys.Error, err))
	}
}

// newFeed creates a new OPDS feed with standard namespaces and common links.
func newFeed(id, title, selfHref, selfType string, updated time.Time) *opdsFeed {
	ts := updated.UTC().Format(time.RFC3339)
	f := &opdsFeed{
		ID:      id,
		Title:   title,
		Updated: ts,
		Author: &opdsAuthor{
			Name: "Biblioteka",
		},
		Links: []opdsLink{
			{Rel: opdsRelSelf, Href: selfHref, Type: selfType},
			{Rel: opdsRelStart, Href: "/opds/", Type: opdsNavigationType},
		},
	}
	return f
}

// bookToEntry converts a db.Book with its authors and files to an OPDS entry.
func bookToEntry(book *db.Book, authors []db.Author, files []db.BookFile) opdsEntry {
	updated := book.UpdatedAt.Time
	if updated.IsZero() {
		updated = book.CreatedAt.Time
	}
	if updated.IsZero() {
		updated = time.Now()
	}

	entry := opdsEntry{
		Title:   book.Title,
		ID:      fmt.Sprintf("urn:uuid:%s", book.ID),
		Updated: updated.UTC().Format(time.RFC3339),
	}

	if book.Description != nil && *book.Description != "" {
		entry.Summary = *book.Description
	}

	if book.Publisher != nil {
		entry.Publisher = *book.Publisher
	}

	if book.Language != nil {
		entry.Language = *book.Language
	}

	if book.PublicationDate != nil {
		entry.Issued = *book.PublicationDate
	}

	if book.ISBN13 != nil {
		entry.Identifier = fmt.Sprintf("urn:isbn:%s", *book.ISBN13)
	} else if book.ISBN10 != nil {
		entry.Identifier = fmt.Sprintf("urn:isbn:%s", *book.ISBN10)
	}

	for _, a := range authors {
		entry.Authors = append(entry.Authors, opdsAuthor{
			Name: a.Name,
			URI:  fmt.Sprintf("/opds/authors/%s", a.ID),
		})
	}

	if book.CoverImageURL != nil && *book.CoverImageURL != "" {
		coverMIME := coverImageMIME(*book.CoverImageURL)
		entry.Links = append(entry.Links, opdsLink{
			Rel:  opdsRelImage,
			Href: *book.CoverImageURL,
			Type: coverMIME,
		})
		entry.Links = append(entry.Links, opdsLink{
			Rel:  opdsRelThumbnail,
			Href: *book.CoverImageURL,
			Type: coverMIME,
		})
	}

	for _, f := range files {
		mimeType := fileTypeMIME(f.FileType)
		entry.Links = append(entry.Links, opdsLink{
			Rel:   opdsRelAcquisition,
			Href:  fmt.Sprintf("/opds/books/%s/download/%s", book.ID, f.ID),
			Type:  mimeType,
			Title: f.FileName,
		})
	}

	return entry
}

// fileTypeMIME returns the MIME type for a given book file type extension.
func fileTypeMIME(fileType string) string {
	switch strings.ToLower(fileType) {
	case "epub":
		return "application/epub+zip"
	case "pdf":
		return "application/pdf"
	case "mobi":
		return "application/x-mobipocket-ebook"
	case "azw", "azw3":
		return "application/vnd.amazon.ebook"
	case "cbz":
		return "application/vnd.comicbook+zip"
	case "cbr":
		return "application/vnd.comicbook-rar"
	case "fb2":
		return "application/x-fictionbook+xml"
	default:
		return "application/octet-stream"
	}
}

// coverImageMIME returns the MIME type for a cover image URL by inspecting
// its file extension. Defaults to "image/jpeg" when the type cannot be determined.
func coverImageMIME(imageURL string) string {
	ext := strings.ToLower(filepath.Ext(imageURL))
	// Strip query strings or fragments that might follow the extension.
	if idx := strings.IndexAny(ext, "?#"); idx != -1 {
		ext = ext[:idx]
	}
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// HandleRoot serves the OPDS root catalog feed.
func (h *OPDSHandler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(ctx, w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	now := time.Now()
	feed := newFeed(
		"tag:biblioteka:catalog",
		"Biblioteka",
		"/opds/",
		opdsNavigationType,
		now,
	)
	feed.Links = append(feed.Links, opdsLink{
		Rel:  opdsRelSearch,
		Href: "/opds/search?q={searchTerms}",
		Type: "application/atom+xml;profile=opds-catalog",
	})

	feed.Entries = []opdsEntry{
		{
			Title:   "All Books",
			ID:      "tag:biblioteka:catalog:books",
			Updated: now.UTC().Format(time.RFC3339),
			Content: &opdsContent{Type: "text", Value: "Browse all books in your library"},
			Links: []opdsLink{
				{Rel: opdsRelSubsection, Href: "/opds/books", Type: opdsAcquisitionType},
			},
		},
		{
			Title:   "By Author",
			ID:      "tag:biblioteka:catalog:authors",
			Updated: now.UTC().Format(time.RFC3339),
			Content: &opdsContent{Type: "text", Value: "Browse books by author"},
			Links: []opdsLink{
				{Rel: opdsRelSubsection, Href: "/opds/authors", Type: opdsNavigationType},
			},
		},
		{
			Title:   "By Series",
			ID:      "tag:biblioteka:catalog:series",
			Updated: now.UTC().Format(time.RFC3339),
			Content: &opdsContent{Type: "text", Value: "Browse books by series"},
			Links: []opdsLink{
				{Rel: opdsRelSubsection, Href: "/opds/series", Type: opdsNavigationType},
			},
		},
	}

	writeOPDS(ctx, w, feed)
}

// HandleBooks serves an OPDS acquisition feed of all books.
func (h *OPDSHandler) HandleBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(ctx, w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	books, err := h.DB.ListBooks(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list books", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to list books")
		return
	}

	// Collect book IDs for batched related-data queries.
	bookIDs := make([]string, 0, len(books))
	for i := range books {
		bookIDs = append(bookIDs, books[i].ID)
	}

	// Batch-load authors for all books.
	authorsByBook, err := h.DB.GetBookAuthorsForBooks(ctx, bookIDs)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to get authors for books batch", slog.Any(otelkeys.Error, err))
		authorsByBook = nil
	}

	// Batch-load files for all books.
	filesByBook, err := h.DB.ListBookFilesForBooks(ctx, bookIDs)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to get files for books batch", slog.Any(otelkeys.Error, err))
		filesByBook = nil
	}

	feed := newFeed(
		"tag:biblioteka:catalog:books",
		"All Books",
		"/opds/books",
		opdsAcquisitionType,
		time.Now(),
	)
	feed.Links = append(feed.Links, opdsLink{Rel: opdsRelUp, Href: "/opds/", Type: opdsNavigationType})

	for i := range books {
		book := &books[i]

		var authors []db.Author
		if authorsByBook != nil {
			if a, ok := authorsByBook[book.ID]; ok {
				authors = a
			}
		}

		var files []db.BookFile
		if filesByBook != nil {
			if f, ok := filesByBook[book.ID]; ok {
				files = f
			}
		}

		feed.Entries = append(feed.Entries, bookToEntry(book, authors, files))
	}

	writeOPDS(ctx, w, feed)
}

// HandleAuthors serves an OPDS navigation feed of all authors.
func (h *OPDSHandler) HandleAuthors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(ctx, w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	authors, err := h.DB.ListAuthors(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list authors", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to list authors")
		return
	}

	now := time.Now()
	feed := newFeed(
		"tag:biblioteka:catalog:authors",
		"Authors",
		"/opds/authors",
		opdsNavigationType,
		now,
	)
	feed.Links = append(feed.Links, opdsLink{Rel: opdsRelUp, Href: "/opds/", Type: opdsNavigationType})

	for _, a := range authors {
		author := a
		updated := author.UpdatedAt.Time
		if updated.IsZero() {
			updated = now
		}
		feed.Entries = append(feed.Entries, opdsEntry{
			Title:   author.Name,
			ID:      fmt.Sprintf("tag:biblioteka:catalog:authors:%s", author.ID),
			Updated: updated.UTC().Format(time.RFC3339),
			Content: &opdsContent{Type: "text", Value: author.Name},
			Links: []opdsLink{
				{Rel: opdsRelSubsection, Href: fmt.Sprintf("/opds/authors/%s", author.ID), Type: opdsAcquisitionType},
			},
		})
	}

	writeOPDS(ctx, w, feed)
}

// HandleAuthorBooks serves an OPDS acquisition feed for books by a specific author.
func (h *OPDSHandler) HandleAuthorBooks(w http.ResponseWriter, r *http.Request, authorID string) {
	ctx := r.Context()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(ctx, w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	author, err := h.DB.GetAuthor(ctx, authorID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(ctx, w, http.StatusNotFound, "author not found")
			return
		}
		slog.ErrorContext(ctx, "OPDS: failed to get author",
			slog.String(otelkeys.AuthorID, authorID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to get author")
		return
	}

	books, err := h.DB.ListBooksByAuthor(ctx, authorID)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list books by author",
			slog.String(otelkeys.AuthorID, authorID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to list books")
		return
	}

	feed := newFeed(
		fmt.Sprintf("tag:biblioteka:catalog:authors:%s", author.ID),
		author.Name,
		fmt.Sprintf("/opds/authors/%s", author.ID),
		opdsAcquisitionType,
		time.Now(),
	)
	feed.Links = append(feed.Links, opdsLink{Rel: opdsRelUp, Href: "/opds/authors", Type: opdsNavigationType})

	for i := range books {
		book := &books[i]
		authors, err := h.DB.GetBookAuthors(ctx, book.ID)
		if err != nil {
			slog.ErrorContext(ctx, "OPDS: failed to get book authors",
				slog.String(otelkeys.BookID, book.ID),
				slog.Any(otelkeys.Error, err),
			)
			authors = nil
		}
		files, err := h.DB.ListBookFiles(ctx, book.ID)
		if err != nil {
			slog.ErrorContext(ctx, "OPDS: failed to get book files",
				slog.String(otelkeys.BookID, book.ID),
				slog.Any(otelkeys.Error, err),
			)
			files = nil
		}
		feed.Entries = append(feed.Entries, bookToEntry(book, authors, files))
	}

	writeOPDS(ctx, w, feed)
}

// HandleSeriesList serves an OPDS navigation feed of all series.
func (h *OPDSHandler) HandleSeriesList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(ctx, w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	seriesList, err := h.DB.ListSeries(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list series",
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to list series")
		return
	}

	now := time.Now()
	feed := newFeed(
		"tag:biblioteka:catalog:series",
		"Series",
		"/opds/series",
		opdsNavigationType,
		now,
	)
	feed.Links = append(feed.Links, opdsLink{Rel: opdsRelUp, Href: "/opds/", Type: opdsNavigationType})

	for _, s := range seriesList {
		series := s
		updated := series.UpdatedAt.Time
		if updated.IsZero() {
			updated = now
		}
		feed.Entries = append(feed.Entries, opdsEntry{
			Title:   series.Name,
			ID:      fmt.Sprintf("tag:biblioteka:catalog:series:%s", series.ID),
			Updated: updated.UTC().Format(time.RFC3339),
			Content: &opdsContent{Type: "text", Value: series.Name},
			Links: []opdsLink{
				{Rel: opdsRelSubsection, Href: fmt.Sprintf("/opds/series/%s", series.ID), Type: opdsAcquisitionType},
			},
		})
	}

	writeOPDS(ctx, w, feed)
}

// HandleSeriesBooks serves an OPDS acquisition feed for books in a specific series.
func (h *OPDSHandler) HandleSeriesBooks(w http.ResponseWriter, r *http.Request, seriesID string) {
	ctx := r.Context()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(ctx, w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	series, err := h.DB.GetSeries(ctx, seriesID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(ctx, w, http.StatusNotFound, "series not found")
			return
		}
		slog.ErrorContext(ctx, "OPDS: failed to get series",
			slog.String(otelkeys.SeriesID, seriesID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to get series")
		return
	}

	books, err := h.DB.ListBooksBySeries(ctx, seriesID)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to list books by series",
			slog.String(otelkeys.SeriesID, seriesID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to list books")
		return
	}

	feed := newFeed(
		fmt.Sprintf("tag:biblioteka:catalog:series:%s", series.ID),
		series.Name,
		fmt.Sprintf("/opds/series/%s", series.ID),
		opdsAcquisitionType,
		time.Now(),
	)
	feed.Links = append(feed.Links, opdsLink{Rel: opdsRelUp, Href: "/opds/series", Type: opdsNavigationType})

	for i := range books {
		book := &books[i]
		authors, err := h.DB.GetBookAuthors(ctx, book.ID)
		if err != nil {
			slog.ErrorContext(ctx, "OPDS: failed to get book authors",
				slog.String(otelkeys.BookID, book.ID),
				slog.Any(otelkeys.Error, err),
			)
			authors = nil
		}
		files, err := h.DB.ListBookFiles(ctx, book.ID)
		if err != nil {
			slog.ErrorContext(ctx, "OPDS: failed to get book files",
				slog.String(otelkeys.BookID, book.ID),
				slog.Any(otelkeys.Error, err),
			)
			files = nil
		}
		feed.Entries = append(feed.Entries, bookToEntry(book, authors, files))
	}

	writeOPDS(ctx, w, feed)
}

// HandleSearch serves an OPDS acquisition feed for search results.
func (h *OPDSHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(ctx, w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeError(ctx, w, http.StatusBadRequest, "search query 'q' is required")
		return
	}

	books, err := h.DB.SearchBooks(ctx, query)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to search books",
			slog.String(otelkeys.Query, query),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to search books")
		return
	}

	encodedQuery := url.QueryEscape(query)

	feed := newFeed(
		fmt.Sprintf("tag:biblioteka:catalog:search:%s", query),
		fmt.Sprintf("Search: %s", query),
		fmt.Sprintf("/opds/search?q=%s", encodedQuery),
		opdsAcquisitionType,
		time.Now(),
	)
	feed.Links = append(feed.Links, opdsLink{Rel: opdsRelUp, Href: "/opds/", Type: opdsNavigationType})

	for i := range books {
		book := &books[i]
		authors, err := h.DB.GetBookAuthors(ctx, book.ID)
		if err != nil {
			slog.ErrorContext(ctx, "OPDS: failed to get book authors",
				slog.String(otelkeys.BookID, book.ID),
				slog.Any(otelkeys.Error, err),
			)
			authors = nil
		}
		files, err := h.DB.ListBookFiles(ctx, book.ID)
		if err != nil {
			slog.ErrorContext(ctx, "OPDS: failed to get book files",
				slog.String(otelkeys.BookID, book.ID),
				slog.Any(otelkeys.Error, err),
			)
			files = nil
		}
		feed.Entries = append(feed.Entries, bookToEntry(book, authors, files))
	}

	writeOPDS(ctx, w, feed)
}

// HandleDownload serves the raw file for a book file download.
func (h *OPDSHandler) HandleDownload(w http.ResponseWriter, r *http.Request, bookID, fileID string) {
	ctx := r.Context()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(ctx, w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	bf, err := h.DB.GetBookFile(ctx, fileID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(ctx, w, http.StatusNotFound, "book file not found")
			return
		}
		slog.ErrorContext(ctx, "OPDS: failed to get book file",
			slog.String(otelkeys.BookFileID, fileID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to get book file")
		return
	}

	// Verify the file belongs to the specified book.
	if bf.BookID != bookID {
		writeError(ctx, w, http.StatusNotFound, "book file not found")
		return
	}

	f, err := os.Open(bf.FilePath)
	if err != nil {
		slog.ErrorContext(ctx, "OPDS: failed to open file",
			slog.String(otelkeys.Path, bf.FilePath),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusNotFound, "file not found on disk")
		return
	}
	defer f.Close()

	mimeType := fileTypeMIME(bf.FileType)
	if mimeType == "application/octet-stream" {
		// Try to detect from file extension.
		if detected := mime.TypeByExtension(filepath.Ext(bf.FileName)); detected != "" {
			mimeType = detected
		}
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, bf.FileName))

	http.ServeContent(w, r, bf.FileName, bf.UpdatedAt.Time, f)
}

// HandleOPDS dispatches all /opds/* routes.
func (h *OPDSHandler) HandleOPDS(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch path {
	case "/opds", "/opds/":
		h.HandleRoot(w, r)
	case "/opds/books":
		h.HandleBooks(w, r)
	case "/opds/authors":
		h.HandleAuthors(w, r)
	case "/opds/series":
		h.HandleSeriesList(w, r)
	case "/opds/search":
		h.HandleSearch(w, r)
	default:
		// /opds/authors/{id}
		if id, ok := extractPathID(path, "/opds/authors/"); ok {
			h.HandleAuthorBooks(w, r, id)
			return
		}
		// /opds/series/{id}
		if id, ok := extractPathID(path, "/opds/series/"); ok {
			h.HandleSeriesBooks(w, r, id)
			return
		}
		// /opds/books/{id}/download/{fileID}
		if bookID, rest, ok := extractPathSegments(path, "/opds/books/"); ok {
			parts := strings.SplitN(rest, "/", 2)
			if len(parts) == 2 && parts[0] == "download" {
				h.HandleDownload(w, r, bookID, parts[1])
				return
			}
		}
		writeError(r.Context(), w, http.StatusNotFound, "not found")
	}
}
