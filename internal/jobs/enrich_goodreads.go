package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/goodreads"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/ptrutil"
	"github.com/amalgamated-tools/biblioteka/internal/pubsub"
)

// JobEnrichGoodreads is the registered name for the Goodreads enrichment job.
const JobEnrichGoodreads = "enrich:goodreads"

// Terminal SSE event types published on the metadata progress channel.
// These constants are shared between the enrichment job (publisher) and the
// SSE handler (consumer) to avoid magic strings.
const (
	EventComplete = "complete"
	EventError    = "error"
	EventNotFound = "not_found"
	EventProgress = "progress"
)

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

// progressEvent is the JSON structure published to Redis pub/sub for SSE clients.
type progressEvent struct {
	Event          string `json:"event"`
	Source         string `json:"source,omitempty"`
	Step           string `json:"step,omitempty"`
	Message        string `json:"message,omitempty"`
	MetadataID     string `json:"metadata_id,omitempty"`
	AIEnrichmentID string `json:"ai_enrichment_id,omitempty"`
}

// NewEnrichGoodreadsHandler returns a worker.Func that fetches Goodreads
// metadata for a book and stores the result as a pending GoodreadsMetadata
// record for user review. If publisher is non-nil, progress events are
// broadcast via Redis pub/sub for SSE clients.
func NewEnrichGoodreadsHandler(database *db.DB, grClient GoodreadsSearcher, publisher pubsub.Publisher) func(ctx context.Context, payload []byte) error {
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

		return enrichGoodreads(ctx, database, grClient, publisher, p)
	}
}

func enrichGoodreads(ctx context.Context, database *db.DB, grClient GoodreadsSearcher, publisher pubsub.Publisher, p EnrichGoodreadsPayload) error {
	if p.BookID == "" || p.UserID == "" {
		return errors.New("enrich goodreads: book_id and user_id are required")
	}

	channel := pubsub.MetadataChannel(p.BookID, p.UserID)

	publishProgress(ctx, publisher, channel, "searching", "Looking up book...")

	book, err := database.GetBook(ctx, p.BookID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch book for Goodreads enrichment",
			slog.String(otelkeys.BookID, p.BookID),
			slog.Any(otelkeys.Error, err),
		)
		publishEvent(ctx, publisher, channel, progressEvent{Event: EventError, Source: db.MetadataSourceGoodreads, Message: "failed to fetch book"})
		return fmt.Errorf("fetch book %s: %w", p.BookID, err)
	}

	result, strategy := lookupGoodreads(ctx, grClient, publisher, channel, book)
	if result == nil {
		slog.InfoContext(ctx, "no Goodreads match found for book",
			slog.String(otelkeys.BookID, p.BookID),
			slog.String(otelkeys.Title, book.Title),
		)
		publishEvent(ctx, publisher, channel, progressEvent{Event: EventNotFound, Source: db.MetadataSourceGoodreads, Message: "No Goodreads match found"})
		return nil
	}

	slog.InfoContext(ctx, "Goodreads match found",
		slog.String(otelkeys.BookID, p.BookID),
		slog.String(otelkeys.LookupStrategy, strategy),
		slog.String(otelkeys.Title, result.BookTitle),
		slog.String(otelkeys.GoodreadsID, result.BookID),
	)

	publishProgress(ctx, publisher, channel, "match_found", fmt.Sprintf("Found: %s", result.BookTitle))

	gm, err := createGoodreadsMetadataFromResult(ctx, database, p.UserID, p.BookID, result)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create goodreads metadata",
			slog.String(otelkeys.BookID, p.BookID),
			slog.Any(otelkeys.Error, err),
		)
		publishEvent(ctx, publisher, channel, progressEvent{Event: EventError, Source: db.MetadataSourceGoodreads, Message: "failed to save metadata"})
		return fmt.Errorf("create goodreads metadata for book %s: %w", p.BookID, err)
	}

	publishEvent(ctx, publisher, channel, progressEvent{Event: EventComplete, Source: db.MetadataSourceGoodreads, MetadataID: gm.ID})

	return nil
}

// lookupGoodreads tries multiple strategies to find a book on Goodreads.
// Returns the first match and the strategy name that found it, or (nil, "")
// if no match is found.
func lookupGoodreads(ctx context.Context, grClient GoodreadsSearcher, publisher pubsub.Publisher, channel string, book *db.Book) (*goodreads.BookResult, string) {
	// Strategy 1: ISBN lookup (highest confidence)
	if isbn := ptrutil.Deref(book.ISBN13); isbn != "" {
		publishProgress(ctx, publisher, channel, "searching_isbn13", "Searching Goodreads by ISBN-13...")
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
	if isbn := ptrutil.Deref(book.ISBN10); isbn != "" {
		publishProgress(ctx, publisher, channel, "searching_isbn10", "Searching Goodreads by ISBN-10...")
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
	if asin := ptrutil.Deref(book.ASIN); asin != "" {
		publishProgress(ctx, publisher, channel, "searching_asin", "Searching Goodreads by ASIN...")
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
	if grID := ptrutil.Deref(book.GoodreadsID); grID != "" {
		publishProgress(ctx, publisher, channel, "searching_goodreads_id", "Searching Goodreads by ID...")
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

	// Strategy 4: Title search (lowest confidence — verify the result title
	// matches before accepting, since a generic search can return unrelated books).
	// Scan up to the first 5 results for a match rather than only checking results[0].
	if book.Title != "" {
		publishProgress(ctx, publisher, channel, "searching_title", "Searching Goodreads by title...")
		results, err := grClient.Search(ctx, book.Title)
		if err == nil && len(results) > 0 {
			const maxTitleCandidates = 5
			limit := min(len(results), maxTitleCandidates)
			for i := range limit {
				if titleSimilar(book.Title, results[i].BookTitle) {
					return &results[i], "title"
				}
			}
			slog.DebugContext(ctx, "Goodreads title search results did not match book title",
				slog.String(otelkeys.BookID, book.ID),
				slog.String(otelkeys.SearchTitle, book.Title),
				slog.Int(otelkeys.Count, limit),
			)
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
		BookID:                  ptrutil.NilIfZero(bookID),
		Title:                   ptrutil.NilIfZero(result.BookTitle),
		ASIN:                    ptrutil.NilIfZero(result.BookASIN),
		ISBN10:                  ptrutil.NilIfZero(result.BookISBN),
		ISBN13:                  ptrutil.NilIfZero(result.BookISBN13),
		GoodreadsID:             ptrutil.NilIfZero(result.BookID),
		Language:                ptrutil.NilIfZero(result.BookLanguage),
		CoverImageURL:           ptrutil.NilIfZero(result.BookImageURL),
		AuthorName:              ptrutil.NilIfZero(result.AuthorName),
		AuthorGoodreadsID:       ptrutil.NilIfZero(result.AuthorID),
		AuthorImageURL:          ptrutil.NilIfZero(result.AuthorProfileImageURL),
		GoodreadsWorkID:         ptrutil.NilIfZero(result.WorkID),
		GoodreadsBookLegacyID:   ptrutil.NilIfZero(result.BookLegacyID),
		GoodreadsWorkLegacyID:   ptrutil.NilIfZero(result.WorkLegacyID),
		GoodreadsAuthorLegacyID: ptrutil.NilIfZero(result.AuthorLegacyID),
	})
}

// titleSimilar checks whether two book titles are similar enough to be
// considered a match. It normalizes both titles to lowercase and checks
// whether one contains the other, which handles common cases like subtitle
// differences ("The Hobbit" vs "The Hobbit, or There and Back Again").
func titleSimilar(a, b string) bool {
	na := strings.ToLower(strings.TrimSpace(a))
	nb := strings.ToLower(strings.TrimSpace(b))
	if na == "" || nb == "" {
		return false
	}
	// Require a minimum length of 4 characters for substring matching to
	// prevent false positives with very short titles (e.g. "It" matching
	// "Digital", "Britain", etc.).
	if na == nb {
		return true
	}
	if len(na) >= 4 && strings.Contains(nb, na) {
		return true
	}
	if len(nb) >= 4 && strings.Contains(na, nb) {
		return true
	}
	return false
}

// publishProgress is a convenience wrapper that publishes a progress event
// with a source of db.MetadataSourceGoodreads.
func publishProgress(ctx context.Context, publisher pubsub.Publisher, channel, step, message string) {
	publishAIProgress(
		ctx,
		publisher,
		channel,
		db.MetadataSourceGoodreads,
		step,
		message,
	)
}

// publishEvent marshals and publishes a progress event to the given channel.
// If publisher is nil or marshaling/publishing fails, the error is logged but
// does not interrupt the job.
func publishEvent(ctx context.Context, publisher pubsub.Publisher, channel string, evt progressEvent) {
	if publisher == nil {
		return
	}
	data, err := json.Marshal(evt)
	if err != nil {
		slog.WarnContext(ctx, "failed to marshal progress event",
			slog.Any(otelkeys.Error, err),
		)
		return
	}
	if err := publisher.Publish(ctx, channel, string(data)); err != nil {
		slog.WarnContext(ctx, "failed to publish progress event",
			slog.String(otelkeys.PubSubChannel, channel),
			slog.Any(otelkeys.Error, err),
		)
	}
}
