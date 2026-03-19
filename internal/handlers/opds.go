package handlers

import (
	"context"
	"encoding/xml"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
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
		writeOPDSError(r, w, http.StatusMethodNotAllowed, opdsNavContentType, "urn:biblioteka:opds:error", "Method not allowed")
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
		writeOPDSError(r, w, http.StatusInternalServerError, opdsAcqContentType, "urn:biblioteka:opds:error", "Failed to read file")
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
