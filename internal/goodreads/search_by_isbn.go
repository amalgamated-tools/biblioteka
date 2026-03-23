package goodreads

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/buger/jsonparser"
)

// SearchByISBN searches Goodreads for books matching the given ISBN and returns a list of search results.
// It uses the Goodreads auto_complete endpoint, which returns a list of books matching the query.
// Since we are searching by ISBN, we expect to get either 0 or 1 results back, but we return a list to be consistent with the other search methods.
func (c *Client) SearchByISBN(ctx context.Context, isbn string) ([]BookResult, error) {
	// make sure that the ISBN is valid before making the HTTP request, to avoid unnecessary requests and to provide better error messages
	if len(isbn) == 0 {
		slog.ErrorContext(
			ctx,
			"ISBN cannot be empty",
		)
		return nil, fmt.Errorf("ISBN cannot be empty")
	}
	// Strip common ISBN formatting before validating and querying Goodreads.
	var normalizedISBN strings.Builder
	for _, r := range isbn {
		switch {
		case r == '-' || r == ' ':
			continue
		case r >= '0' && r <= '9':
			normalizedISBN.WriteRune(r)
		case (r == 'x' || r == 'X') && normalizedISBN.Len() == 9:
			normalizedISBN.WriteRune('X')
		default:
			slog.ErrorContext(ctx, "invalid character in ISBN", slog.String(otelkeys.ISBN, isbn))
			return nil, fmt.Errorf("invalid ISBN: %s (unexpected character %q)", isbn, r)
		}
	}
	isbnValue := normalizedISBN.String()
	if len(isbnValue) != 10 && len(isbnValue) != 13 {
		slog.ErrorContext(
			ctx,
			"invalid ISBN length",
			slog.String(otelkeys.ISBN, isbn),
		)
		return nil, fmt.Errorf("invalid ISBN: %s", isbn)
	}
	if len(isbnValue) == 10 && !ValidISBN10CheckDigit(isbnValue) {
		slog.ErrorContext(
			ctx,
			"invalid ISBN-10 check digit",
			slog.String(otelkeys.ISBN, isbn),
		)
		return nil, fmt.Errorf("invalid ISBN-10 check digit: %s", isbn)
	}
	if len(isbnValue) == 13 && !ValidISBN13CheckDigit(isbnValue) {
		slog.ErrorContext(
			ctx,
			"invalid ISBN-13 check digit",
			slog.String(otelkeys.ISBN, isbn),
		)
		return nil, fmt.Errorf("invalid ISBN-13 check digit: %s", isbn)
	}

	searchURL := fmt.Sprintf("https://goodreads.com/book/auto_complete?format=json&q=%s", url.QueryEscape(isbnValue))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to create HTTP request for Goodreads ISBN search",
			slog.String(otelkeys.Query, isbn),
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.URL, searchURL),
		)
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"HTTP request failed for Goodreads ISBN search",
			slog.String(otelkeys.Query, isbn),
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.URL, searchURL),
		)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(
			ctx,
			"goodreads ISBN search returned non-OK status",
			slog.String(otelkeys.Query, isbn),
			slog.Int(otelkeys.StatusCode, resp.StatusCode),
			slog.String(otelkeys.URL, searchURL),
		)
		return nil, fmt.Errorf("goodreads ISBN search returned status %d", resp.StatusCode)
	}

	const maxResponseSize = 1 << 20 // 1 MB
	bodyText, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to read response body for Goodreads ISBN search",
			slog.String(otelkeys.Query, isbn),
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.URL, searchURL),
		)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if len(bodyText) > maxResponseSize {
		slog.ErrorContext(
			ctx,
			"response body too large for Goodreads ISBN search",
			slog.String(otelkeys.Query, isbn),
			slog.String(otelkeys.URL, searchURL),
		)
		return nil, fmt.Errorf("goodreads ISBN search response too large (exceeded %d bytes)", maxResponseSize)
	}

	results, err := c.parseISBNSearchResponse(ctx, bodyText)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to parse Goodreads ISBN search response",
			slog.String(otelkeys.Query, isbn),
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.URL, searchURL),
		)
		return nil, fmt.Errorf("failed to parse Goodreads ISBN search response: %w", err)
	}

	return results, nil
}

// autocompleteEntry holds the raw data parsed from a single Goodreads
// auto_complete JSON array element before any GraphQL enrichment.
type autocompleteEntry struct {
	bookID     int64
	workID     int64
	title      string
	imageURL   string
	numPages   int64
	authorID   int64
	authorName string
}

// parseISBNSearchResponse parses the JSON response from the Goodreads auto_complete endpoint
// and enriches results via concurrent GraphQL lookups.
func (c *Client) parseISBNSearchResponse(ctx context.Context, bodyText []byte) ([]BookResult, error) {
	// Phase 1: Parse all entries from the JSON array without making any network calls.
	entries, err := parseAutocompleteEntries(ctx, bodyText)
	if err != nil {
		// If the context was cancelled, surface that directly instead of
		// masking it as a JSON-parse error.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		slog.ErrorContext(
			ctx,
			"failed to parse Goodreads ISBN search response as JSON array",
			slog.Any(otelkeys.Error, err),
		)
		return nil, fmt.Errorf("parse Goodreads ISBN search response as JSON array: %w", err)
	}

	if len(entries) == 0 {
		slog.DebugContext(ctx, "parsed Goodreads ISBN search response", slog.Int(otelkeys.Count, 0))
		return []BookResult{}, nil
	}

	// Phase 2: Enrich entries via concurrent GraphQL lookups.
	type indexedResult struct {
		index  int
		result BookResult
	}

	resultsCh := make(chan indexedResult, len(entries))
	var wg sync.WaitGroup

	for i, entry := range entries {
		wg.Add(1)
		go func(idx int, e autocompleteEntry) {
			defer wg.Done()

			// Check context before making a network call.
			if ctx.Err() != nil {
				return
			}

			result, err := c.GetBookByLegacyID(ctx, e.bookID)
			if err == nil {
				resultsCh <- indexedResult{index: idx, result: *result}
				return
			}

			// If context was cancelled, don't fall back to partial data.
			if ctx.Err() != nil {
				return
			}

			// Fallback: build a partial result from the autocomplete data.
			slog.ErrorContext(
				ctx,
				"failed to get book by legacy ID, returning partial result",
				slog.Int64(otelkeys.BookLegacyID, e.bookID),
				slog.Any(otelkeys.Error, err),
			)
			resultsCh <- indexedResult{index: idx, result: buildFallbackResult(e)}
		}(i, entry)
	}

	wg.Wait()
	close(resultsCh)

	// Check context once more after all goroutines finish.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Collect results and restore original order.
	collected := make([]indexedResult, 0, len(entries))
	for ir := range resultsCh {
		collected = append(collected, ir)
	}
	// Sort by original index to produce deterministic output.
	results := make([]BookResult, 0, len(collected))
	ordered := make(map[int]BookResult, len(collected))
	for _, ir := range collected {
		ordered[ir.index] = ir.result
	}
	for i := range entries {
		if r, ok := ordered[i]; ok {
			results = append(results, r)
		}
	}

	slog.DebugContext(
		ctx,
		"parsed Goodreads ISBN search response",
		slog.Int(otelkeys.Count, len(results)),
	)
	return results, nil
}

// parseAutocompleteEntries extracts structured entries from the Goodreads
// auto_complete JSON response without making any network calls.
func parseAutocompleteEntries(ctx context.Context, bodyText []byte) ([]autocompleteEntry, error) {
	var entries []autocompleteEntry

	_, err := jsonparser.ArrayEach(bodyText, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err != nil {
			slog.ErrorContext(ctx, "failed to parse Goodreads ISBN search result", slog.Any(otelkeys.Error, err))
			return
		}

		bookIDStr, err := jsonparser.GetString(value, "bookId")
		if err != nil {
			slog.ErrorContext(ctx, "failed to get bookId from Goodreads ISBN search result", slog.Any(otelkeys.Error, err))
			return
		}

		bookID, err := strconv.ParseInt(bookIDStr, 10, 64)
		if err != nil {
			slog.ErrorContext(ctx, "failed to parse bookId from Goodreads ISBN search result", slog.Any(otelkeys.Error, err))
			return
		}

		workIDStr, err := jsonparser.GetString(value, "workId")
		if err != nil {
			slog.ErrorContext(ctx, "failed to get workId from Goodreads ISBN search result", slog.Any(otelkeys.Error, err))
			return
		}

		workID, err := strconv.ParseInt(workIDStr, 10, 64)
		if err != nil {
			slog.ErrorContext(ctx, "failed to parse workId from Goodreads ISBN search result", slog.Any(otelkeys.Error, err))
			return
		}

		title, err := jsonparser.GetString(value, "title")
		if err != nil {
			slog.ErrorContext(ctx, "failed to get title from Goodreads ISBN search result", slog.Any(otelkeys.Error, err))
			return
		}

		// Guard against semantically invalid zero IDs and empty titles.
		if workID == 0 || bookID == 0 || title == "" {
			slog.ErrorContext(
				ctx,
				"missing required fields in Goodreads ISBN search result",
				slog.Int64(otelkeys.BookLegacyID, bookID),
				slog.Int64(otelkeys.WorkLegacyID, workID),
				slog.String(otelkeys.Title, title),
			)
			return
		}

		imageURL, err := jsonparser.GetString(value, "imageUrl")
		if err != nil {
			slog.DebugContext(ctx, "missing imageUrl in Goodreads search result", slog.Any(otelkeys.Error, err))
		}

		numPages, err := jsonparser.GetInt(value, "numPages")
		if err != nil {
			slog.DebugContext(ctx, "missing numPages in Goodreads search result", slog.Any(otelkeys.Error, err))
		}

		authorID, err := jsonparser.GetInt(value, "author", "id")
		if err != nil {
			slog.DebugContext(ctx, "missing author ID in Goodreads search result", slog.Any(otelkeys.Error, err))
		}

		authorName, err := jsonparser.GetString(value, "author", "name")
		if err != nil {
			slog.DebugContext(ctx, "missing author name in Goodreads search result", slog.Any(otelkeys.Error, err))
		}

		entries = append(entries, autocompleteEntry{
			bookID:     bookID,
			workID:     workID,
			title:      title,
			imageURL:   imageURL,
			numPages:   numPages,
			authorID:   authorID,
			authorName: authorName,
		})
	})

	return entries, err
}

// buildFallbackResult creates a BookResult from autocomplete data when the
// GraphQL enrichment call fails.
func buildFallbackResult(e autocompleteEntry) BookResult {
	// NOTE: fallback path only has legacy integer IDs; BookID and WorkID will be
	// integer strings (e.g. "123"), not KCA URIs (e.g. "kca://book/...").
	// Callers must not assume the same format as the primary GraphQL path.
	return BookResult{
		BookID:            strconv.FormatInt(e.bookID, 10),
		WorkID:            strconv.FormatInt(e.workID, 10),
		BookImageURL:      e.imageURL,
		BookLegacyID:      e.bookID,
		WorkLegacyID:      e.workID,
		BookTitle:         e.title,
		BookNumberOfPages: e.numPages,
		AuthorLegacyID:    e.authorID,
		AuthorName:        e.authorName,
	}
}
