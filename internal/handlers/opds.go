package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	opdspkg "github.com/amalgamated-tools/biblioteka/internal/opds"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// OPDSHandler serves OPDS catalog feeds.
type OPDSHandler struct {
	DB *db.DB
}

// HandleOPDS dispatches OPDS requests based on the URL path.
func (h *OPDSHandler) HandleOPDS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		writeOPDSError(r, w, http.StatusMethodNotAllowed, opdspkg.NavContentType, "urn:biblioteka:opds:error", "Method not allowed")
		return
	}

	subPath := strings.TrimPrefix(r.URL.Path, "/opds")
	subPath = strings.TrimSuffix(subPath, "/")

	switch {
	case subPath == "" || subPath == "/":
		h.rootFeed(w, r)
	case subPath == "/all":
		h.allBooks(w, r)
	case subPath == "/recent":
		h.recentBooks(w, r)
	case subPath == "/authors":
		h.authorsFeed(w, r)
	case strings.HasPrefix(subPath, "/authors/"):
		h.authorBooks(w, r, strings.TrimPrefix(subPath, "/authors/"))
	case subPath == "/series":
		h.seriesFeed(w, r)
	case strings.HasPrefix(subPath, "/series/"):
		h.seriesBooks(w, r, strings.TrimPrefix(subPath, "/series/"))
	case subPath == "/search":
		if r.URL.Query().Get("q") != "" {
			h.searchResults(w, r)
		} else {
			h.openSearchDescription(w, r)
		}
	case strings.HasPrefix(subPath, "/download/"):
		h.downloadFile(w, r, strings.TrimPrefix(subPath, "/download/"))
	case strings.HasPrefix(subPath, "/covers/"):
		h.serveCover(w, r, strings.TrimPrefix(subPath, "/covers/"))
	default:
		http.NotFound(w, r)
	}
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
		XMLNS:       opdspkg.XMLNSOpenSearch,
		ShortName:   "Biblioteka",
		Description: "Search books in Biblioteka",
		URL: osURL{
			Type:     opdspkg.AcqContentType,
			Template: baseURL + "/search?q={searchTerms}",
		},
	}

	w.Header().Set("Content-Type", opdspkg.SearchType)
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

	mimeType := opdspkg.FileTypeMIME[strings.ToLower(bf.FileType)]
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
		writeOPDSError(r, w, http.StatusInternalServerError, opdspkg.AcqContentType, "urn:biblioteka:opds:error", "Failed to read file")
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": bf.FileName}))
	http.ServeContent(w, r, bf.FileName, stat.ModTime(), f)
}

func (h *OPDSHandler) serveCover(w http.ResponseWriter, r *http.Request, bookID string) {
	ctx := r.Context()

	book, err := h.DB.GetBook(ctx, bookID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(ctx, "OPDS: failed to fetch book for cover",
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
		if len(data) > 0 {
			sniffed := http.DetectContentType(data)
			declaredIsImage := strings.HasPrefix(contentType, "image/")
			sniffedIsImage := strings.HasPrefix(sniffed, "image/")
			switch {
			case sniffedIsImage:
				// Trust the sniffed image type when it is also an image.
				contentType = sniffed
			case declaredIsImage:
				// Trust the declared image MIME type even if sniffing disagrees
				// (e.g. SVG detected as text/xml or application/octet-stream).
			default:
				http.Error(w, "invalid cover image", http.StatusInternalServerError)
				return
			}
		}
		if !strings.HasPrefix(contentType, "image/") {
			http.Error(w, "invalid cover image", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", contentType)
		http.ServeContent(w, r, "cover", book.UpdatedAt.Time, bytes.NewReader(data))
		return
	}
	if !errors.Is(err, errNotDataURL) {
		http.Error(w, "invalid cover image", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, *book.CoverImageURL, http.StatusTemporaryRedirect)
}

// bookEntries converts a slice of books into OPDS entry elements, including
// authors and download links for each book. Uses batch queries to avoid N+1.
func (h *OPDSHandler) bookEntries(ctx context.Context, books []db.Book, baseURL string) []opdspkg.Entry {
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

	entries := make([]opdspkg.Entry, 0, len(books))
	for _, book := range books {
		entry := opdspkg.Entry{
			Title:   book.Title,
			ID:      baseURL + "/books/" + book.ID,
			Updated: book.UpdatedAt.Format(time.RFC3339),
		}

		if book.Description != nil && *book.Description != "" {
			entry.Content = &opdspkg.Content{Type: "text", Value: *book.Description}
		}

		// Add authors from batch result.
		if authorsByBook != nil {
			for _, a := range authorsByBook[book.ID] {
				entry.Authors = append(entry.Authors, opdspkg.Author{Name: a.Name})
			}
		}

		// Add cover image link
		if book.CoverImageURL != nil && *book.CoverImageURL != "" {
			coverURL := *book.CoverImageURL
			coverType := coverMIMEType(coverURL)
			// Data URLs must not be inlined — point to the cover endpoint instead.
			if strings.HasPrefix(coverURL, "data:") {
				coverURL = baseURL + "/covers/" + book.ID
			}
			entry.Links = append(entry.Links, opdspkg.Link{
				Rel:  opdspkg.RelImage,
				Href: coverURL,
				Type: coverType,
			})
		}

		// Add download links from batch result.
		if filesByBook != nil {
			for _, f := range filesByBook[book.ID] {
				mimeType := opdspkg.FileTypeMIME[strings.ToLower(f.FileType)]
				if mimeType == "" {
					mimeType = "application/octet-stream"
				}
				entry.Links = append(entry.Links, opdspkg.Link{
					Rel:  opdspkg.RelAcquisition,
					Href: baseURL + "/download/" + f.ID,
					Type: mimeType,
				})
			}
		}

		entries = append(entries, entry)
	}
	return entries
}
