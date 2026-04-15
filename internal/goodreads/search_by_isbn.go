package goodreads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/buger/jsonparser"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// SearchByISBN searches Goodreads for books matching the given ISBN and returns a list of search results.
// Returns a list for consistency with Search, but typically yields 0 or 1 results.
func (c *Client) SearchByISBN(ctx context.Context, isbn string) ([]BookResult, error) {
	// make sure that the ISBN is valid before making the HTTP request, to avoid unnecessary requests and to provide better error messages
	if len(isbn) == 0 {
		return nil, errors.New("ISBN cannot be empty")
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
			return nil, fmt.Errorf("invalid ISBN: %s (unexpected character %q)", isbn, r)
		}
	}
	isbnValue := normalizedISBN.String()
	if len(isbnValue) != 10 && len(isbnValue) != 13 {
		return nil, fmt.Errorf("invalid ISBN: %s", isbn)
	}
	if len(isbnValue) == 10 && !ValidISBN10CheckDigit(isbnValue) {
		return nil, fmt.Errorf("invalid ISBN-10 check digit: %s", isbn)
	}
	if len(isbnValue) == 13 && !ValidISBN13CheckDigit(isbnValue) {
		return nil, fmt.Errorf("invalid ISBN-13 check digit: %s", isbn)
	}

	searchURL := fmt.Sprintf("https://goodreads.com/book/auto_complete?format=json&q=%s", url.QueryEscape(isbnValue))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Goodreads ISBN search request for ISBN %q (%s): %w", isbn, searchURL, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("goodreads ISBN search request failed for ISBN %q (%s): %w", isbn, searchURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("goodreads ISBN search for ISBN %q (%s) returned status %d (%s)", isbn, searchURL, resp.StatusCode, resp.Status)
	}

	const maxResponseSize = 1 << 20 // 1 MB
	bodyText, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for ISBN %q (%s): %w", isbn, searchURL, err)
	}
	if len(bodyText) > maxResponseSize {
		return nil, fmt.Errorf("goodreads ISBN search response for ISBN %q (%s) too large (exceeded %d bytes)", isbn, searchURL, maxResponseSize)
	}

	results, err := c.parseISBNSearchResponse(ctx, bodyText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Goodreads ISBN search response for ISBN %q (%s): %w", isbn, searchURL, err)
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
	authorID   int64
	authorName string
}

// parseISBNSearchResponse parses the JSON response from the Goodreads auto_complete endpoint
// and enriches results via concurrent GraphQL lookups.
func (c *Client) parseISBNSearchResponse(ctx context.Context, bodyText []byte) ([]BookResult, error) {
	// Phase 1: Parse all entries from the JSON array without making any network calls.
	entries, err := parseAutocompleteEntries(bodyText)
	if err != nil {
		// If the context was cancelled, surface that directly instead of
		// masking it as a JSON-parse error.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("parse Goodreads ISBN search response as JSON array: %w", err)
	}

	if len(entries) == 0 {
		return []BookResult{}, nil
	}

	// Cap results to avoid excessive outbound requests; the top entries from
	// Goodreads autocomplete are the most relevant.
	const maxResults = 5
	if len(entries) > maxResults {
		entries = entries[:maxResults]
	}

	// Phase 2: Enrich entries via concurrent GraphQL lookups.
	type indexedResult struct {
		index  int
		result BookResult
	}

	// Bail early if context is already done before spinning up goroutines.
	if ctx.Err() != nil {
		return nil, ctx.Err()
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
			slog.WarnContext(ctx, "failed to get book by legacy ID, returning partial result",
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
	ordered := make(map[int]BookResult, len(entries))
	for ir := range resultsCh {
		ordered[ir.index] = ir.result
	}
	results := make([]BookResult, 0, len(ordered))
	for i := range entries {
		if r, ok := ordered[i]; ok {
			results = append(results, r)
		}
	}

	return results, nil
}

// parseAutocompleteEntries extracts structured entries from the Goodreads
// auto_complete JSON response without making any network calls.
func parseAutocompleteEntries(bodyText []byte) ([]autocompleteEntry, error) {
	var entries []autocompleteEntry

	_, err := jsonparser.ArrayEach(bodyText, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err != nil {
			return
		}

		bookIDStr, err := jsonparser.GetString(value, "bookId")
		if err != nil {
			return
		}

		bookID, err := strconv.ParseInt(bookIDStr, 10, 64)
		if err != nil {
			return
		}

		workIDStr, err := jsonparser.GetString(value, "workId")
		if err != nil {
			return
		}

		workID, err := strconv.ParseInt(workIDStr, 10, 64)
		if err != nil {
			return
		}

		title, err := jsonparser.GetString(value, "title")
		if err != nil {
			return
		}

		// Guard against semantically invalid zero IDs and empty titles.
		if workID == 0 || bookID == 0 || title == "" {
			return
		}

		imageURL, _ := jsonparser.GetString(value, "imageUrl")
		authorID, _ := jsonparser.GetInt(value, "author", "id")
		authorName, _ := jsonparser.GetString(value, "author", "name")

		entries = append(entries, autocompleteEntry{
			bookID:     bookID,
			workID:     workID,
			title:      title,
			imageURL:   imageURL,
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
		BookID:         strconv.FormatInt(e.bookID, 10),
		WorkID:         strconv.FormatInt(e.workID, 10),
		BookImageURL:   e.imageURL,
		BookLegacyID:   e.bookID,
		WorkLegacyID:   e.workID,
		BookTitle:      e.title,
		AuthorLegacyID: e.authorID,
		AuthorName:     e.authorName,
	}
}
