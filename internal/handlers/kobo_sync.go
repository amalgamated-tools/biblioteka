package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/kobo"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// HandleSync handles GET /v1/library/sync.
func (h *KoboHandler) HandleSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	tokenValue := auth.KoboTokenFromContext(ctx)

	syncToken := kobo.ParseSyncToken(r.Header.Get("x-kobo-synctoken"))
	slog.DebugContext(ctx, "kobo library sync request", slog.String(otelkeys.UserID, userID))

	// Fetch one more than the page size to detect whether there are more results.
	books, err := h.DB.ListBooksModifiedSince(ctx, syncToken.BooksLastModified, syncToken.BooksLastID, kobo.SyncPageSize+1)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list books for kobo sync", slog.Any(otelkeys.Error, err))
		writeKoboJSON(w, http.StatusInternalServerError, []any{})
		return
	}

	hasMore := len(books) > kobo.SyncPageSize
	if hasMore {
		books = books[:kobo.SyncPageSize]
	}

	// Batch-load authors, files, and series for the returned books.
	bookIDs := make([]string, len(books))
	for i, b := range books {
		bookIDs[i] = b.ID
	}

	authorsByBook, err := h.DB.GetAuthorsForBooks(ctx, bookIDs)
	if err != nil {
		slog.WarnContext(ctx, "failed to batch-load authors for kobo sync", slog.Any(otelkeys.Error, err))
	}
	filesByBook, err := h.DB.GetFilesForBooks(ctx, bookIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to batch-load files for kobo sync", slog.Any(otelkeys.Error, err))
		writeKoboJSON(w, http.StatusInternalServerError, []any{})
		return
	}
	seriesByBook, err := h.DB.GetSeriesForBooks(ctx, bookIDs)
	if err != nil {
		slog.WarnContext(ctx, "failed to batch-load series for kobo sync", slog.Any(otelkeys.Error, err))
	}

	// Batch-load reading states for the current page's books.
	readingStatesByBook, err := h.DB.GetReadingStatesForBooks(ctx, userID, bookIDs, syncToken.ReadingStateLastModified)
	if err != nil {
		slog.WarnContext(ctx, "failed to batch-load reading states for kobo sync", slog.Any(otelkeys.Error, err))
	}
	if readingStatesByBook == nil {
		readingStatesByBook = make(map[string]*db.KoboReadingState)
	}

	base := schemeAndHost(r)
	var newBooksLastModified time.Time
	var newBooksLastID string
	newReadingStateLastModified := syncToken.ReadingStateLastModified

	syncResults := make([]any, 0, len(books))
	for _, book := range books {
		bk := book // copy
		files := filesByBook[bk.ID]
		authors := authorsByBook[bk.ID]
		series := seriesByBook[bk.ID]

		downloadURLs := kobo.DownloadURLs(base, tokenValue, bk.ID, files)

		// Always advance the high-water mark so that books without downloadable
		// files do not stall pagination — otherwise the next sync would receive
		// the same page again if all its books were file-less.
		if bk.UpdatedAt.After(newBooksLastModified) || (bk.UpdatedAt.Equal(newBooksLastModified) && bk.ID > newBooksLastID) {
			newBooksLastModified = bk.UpdatedAt.Time
			newBooksLastID = bk.ID
		}

		if len(downloadURLs) == 0 {
			// Skip books that have no downloadable files.
			continue
		}

		entitlement := map[string]any{
			"BookEntitlement": kobo.BookEntitlement(&bk),
			"BookMetadata":    kobo.BookMetadata(&bk, authors, series, downloadURLs),
		}

		if rs, ok := readingStatesByBook[bk.ID]; ok {
			if rs.UpdatedAt.After(syncToken.ReadingStateLastModified) {
				entitlement["ReadingState"] = kobo.ReadingStateResponse(rs)
				if rs.UpdatedAt.After(newReadingStateLastModified) {
					newReadingStateLastModified = rs.UpdatedAt.Time
				}
			}
		}

		if bk.CreatedAt.After(syncToken.BooksLastModified) {
			syncResults = append(syncResults, map[string]any{"NewEntitlement": entitlement})
		} else {
			syncResults = append(syncResults, map[string]any{"ChangedEntitlement": entitlement})
		}
	}

	newSyncToken := kobo.SyncToken{
		BooksLastModified:        syncToken.BooksLastModified,
		BooksLastID:              syncToken.BooksLastID,
		ReadingStateLastModified: newReadingStateLastModified,
	}
	if newBooksLastModified.After(syncToken.BooksLastModified) || (newBooksLastModified.Equal(syncToken.BooksLastModified) && newBooksLastID > syncToken.BooksLastID) {
		newSyncToken.BooksLastModified = newBooksLastModified
		newSyncToken.BooksLastID = newBooksLastID
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("x-kobo-synctoken", kobo.EncodeSyncToken(newSyncToken))
	if hasMore {
		w.Header().Set("x-kobo-sync", "continue")
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(syncResults)
}
