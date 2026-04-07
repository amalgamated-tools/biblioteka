package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/goodreads"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// JobEnrichGoodreads is the registered name for the Goodreads enrichment job.
const JobEnrichGoodreads = "enrich:goodreads"

// EnrichGoodreadsPayload is the JSON payload for the enrich:goodreads job.
type EnrichGoodreadsPayload struct {
	BookID string `json:"book_id"`
	UserID string `json:"user_id"`
}

// GoodreadsSearcher abstracts the Goodreads client methods used by the
// enrichment job, allowing test doubles to be injected.
type GoodreadsSearcher interface {
	Search(ctx context.Context, query string) ([]goodreads.BookResult, error)
	SearchByISBN(ctx context.Context, isbn string) ([]goodreads.BookResult, error)
	GetBookByASIN(ctx context.Context, asin string) (*goodreads.BookResult, error)
	GetBookByID(ctx context.Context, grID string) (*goodreads.BookResult, error)
}

// NewEnrichGoodreadsHandler returns a worker.Func that fetches Goodreads
// metadata for a book and stores the result as a pending GoodreadsMetadata
// record for user review.
func NewEnrichGoodreadsHandler(database *db.DB, grClient GoodreadsSearcher) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, payload []byte) error {
		var p EnrichGoodreadsPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			slog.ErrorContext(ctx, "failed to unmarshal enrich:goodreads payload",
				slog.Any(otelkeys.Error, err),
			)
			return fmt.Errorf("unmarshal enrich goodreads payload: %w", err)
		}

		slog.DebugContext(ctx, "enrich:goodreads job received",
			slog.String(otelkeys.BookID, p.BookID),
			slog.String(otelkeys.UserID, p.UserID),
		)

		return enrichGoodreads(ctx, database, grClient, p)
	}
}

func enrichGoodreads(ctx context.Context, database *db.DB, grClient GoodreadsSearcher, p EnrichGoodreadsPayload) error {
	if p.BookID == "" || p.UserID == "" {
		return errors.New("enrich goodreads: book_id and user_id are required")
	}

	book, err := database.GetBook(ctx, p.BookID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch book for Goodreads enrichment",
			slog.String(otelkeys.BookID, p.BookID),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("fetch book %s: %w", p.BookID, err)
	}

	result, strategy := lookupGoodreads(ctx, grClient, book)
	if result == nil {
		slog.InfoContext(ctx, "no Goodreads match found for book",
			slog.String(otelkeys.BookID, p.BookID),
			slog.String(otelkeys.Title, book.Title),
		)
		return nil
	}

	slog.InfoContext(ctx, "Goodreads match found",
		slog.String(otelkeys.BookID, p.BookID),
		slog.String(otelkeys.LookupStrategy, strategy),
		slog.String(otelkeys.Title, result.BookTitle),
		slog.String(otelkeys.GoodreadsID, result.BookID),
	)

	if _, err := createGoodreadsMetadataFromResult(ctx, database, p.UserID, p.BookID, result); err != nil {
		slog.ErrorContext(ctx, "failed to create goodreads metadata",
			slog.String(otelkeys.BookID, p.BookID),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("create goodreads metadata for book %s: %w", p.BookID, err)
	}

	return nil
}

// lookupGoodreads tries multiple strategies to find a book on Goodreads.
// Returns the first match and the strategy name that found it, or (nil, "")
// if no match is found.
func lookupGoodreads(ctx context.Context, grClient GoodreadsSearcher, book *db.Book) (*goodreads.BookResult, string) {
	// Strategy 1: ISBN lookup (highest confidence)
	if isbn := derefStr(book.ISBN13); isbn != "" {
		results, err := grClient.SearchByISBN(ctx, isbn)
		if err == nil && len(results) > 0 {
			return &results[0], "isbn13"
		}
		if err != nil {
			slog.WarnContext(ctx, "Goodreads ISBN13 lookup failed",
				slog.String(otelkeys.BookID, book.ID),
				slog.Any(otelkeys.Error, err),
			)
		}
	}
	if isbn := derefStr(book.ISBN10); isbn != "" {
		results, err := grClient.SearchByISBN(ctx, isbn)
		if err == nil && len(results) > 0 {
			return &results[0], "isbn10"
		}
		if err != nil {
			slog.WarnContext(ctx, "Goodreads ISBN10 lookup failed",
				slog.String(otelkeys.BookID, book.ID),
				slog.Any(otelkeys.Error, err),
			)
		}
	}

	// Strategy 2: ASIN lookup
	if asin := derefStr(book.ASIN); asin != "" {
		result, err := grClient.GetBookByASIN(ctx, asin)
		if err == nil && result != nil {
			return result, "asin"
		}
		if err != nil {
			slog.WarnContext(ctx, "Goodreads ASIN lookup failed",
				slog.String(otelkeys.BookID, book.ID),
				slog.Any(otelkeys.Error, err),
			)
		}
	}

	// Strategy 3: Goodreads ID lookup
	if grID := derefStr(book.GoodreadsID); grID != "" {
		result, err := grClient.GetBookByID(ctx, grID)
		if err == nil && result != nil {
			return result, "goodreads_id"
		}
		if err != nil {
			slog.WarnContext(ctx, "Goodreads ID lookup failed",
				slog.String(otelkeys.BookID, book.ID),
				slog.Any(otelkeys.Error, err),
			)
		}
	}

	// Strategy 4: Title search (lowest confidence)
	if book.Title != "" {
		results, err := grClient.Search(ctx, book.Title)
		if err == nil && len(results) > 0 {
			return &results[0], "title"
		}
		if err != nil {
			slog.WarnContext(ctx, "Goodreads title search failed",
				slog.String(otelkeys.BookID, book.ID),
				slog.Any(otelkeys.Error, err),
			)
		}
	}

	return nil, ""
}

// createGoodreadsMetadataFromResult maps a Goodreads BookResult to a
// GoodreadsMetadata record and persists it with "pending" status.
func createGoodreadsMetadataFromResult(ctx context.Context, database *db.DB, userID, bookID string, result *goodreads.BookResult) (*db.GoodreadsMetadata, error) {
	return database.CreateGoodreadsMetadata(ctx, userID, db.GoodreadsMetadataInput{
		BookID:                  strPtr(bookID),
		Title:                   strPtr(result.BookTitle),
		ASIN:                    strPtr(result.BookASIN),
		ISBN10:                  strPtr(result.BookISBN),
		ISBN13:                  strPtr(result.BookISBN13),
		GoodreadsID:             strPtr(result.BookID),
		Language:                strPtr(result.BookLanguage),
		CoverImageURL:           strPtr(result.BookImageURL),
		AuthorName:              strPtr(result.AuthorName),
		AuthorGoodreadsID:       strPtr(result.AuthorID),
		AuthorImageURL:          strPtr(result.AuthorProfileImageURL),
		GoodreadsWorkID:         strPtr(result.WorkID),
		GoodreadsBookLegacyID:   int64Ptr(result.BookLegacyID),
		GoodreadsWorkLegacyID:   int64Ptr(result.WorkLegacyID),
		GoodreadsAuthorLegacyID: int64Ptr(result.AuthorLegacyID),
	})
}
