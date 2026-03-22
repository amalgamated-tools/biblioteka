package goodreads

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/buger/jsonparser"
)

// SearchByISBN searches Goodreads for books matching the given ISBN and returns a list of search results.
// It uses the Goodreads auto_complete endpoint, which returns a list of books matching the query. We then filter
// the results to only include those that have a matching ISBN.
func (c *Client) SearchByISBN(ctx context.Context, isbn string) ([]BookResult, error) {
	searchURL := fmt.Sprintf("https://goodreads.com/book/auto_complete?format=json&q=%s", url.QueryEscape(isbn))
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to create HTTP request for Goodreads ISBN search",
			slog.Any(otelkeys.Query, isbn),
			slog.Any(otelkeys.Error, err),
			slog.Any(otelkeys.URL, searchURL),
		)
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"HTTP request failed for Goodreads ISBN search",
			slog.Any(otelkeys.Query, isbn),
			slog.Any(otelkeys.Error, err),
			slog.Any(otelkeys.URL, searchURL),
		)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("goodreads ISBN search returned status %d", resp.StatusCode)
	}

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to read response body for Goodreads ISBN search",
			slog.Any(otelkeys.Query, isbn),
			slog.Any(otelkeys.Error, err),
			slog.Any(otelkeys.URL, searchURL),
		)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return c.parseISBNSearchResponse(ctx, bodyText), nil
}

// parseISBNSearchResponse parses the JSON response from the Goodreads auto_complete endpoint.
func (c *Client) parseISBNSearchResponse(ctx context.Context, bodyText []byte) []BookResult {
	results := make([]BookResult, 0)

	_, err := jsonparser.ArrayEach(bodyText, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err != nil {
			slog.ErrorContext(
				ctx,
				"failed to parse Goodreads ISBN search result",
				slog.Any(otelkeys.Error, err),
			)
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

		// when we hit here, we have a valid book id, so we can hit the graphql endpoint
		// to get the rest of the data we need for the search result
		result, err := c.GetBookByLegacyID(ctx, bookID)
		if err == nil {
			results = append(results, *result)
			return
		}

		// if we fail to get the book details, we can still return a partial result with the data we have
		slog.ErrorContext(
			ctx,
			"failed to get book details for Goodreads ISBN search result, returning partial result",
			slog.Any(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)

		imageURL, err := jsonparser.GetString(value, "imageUrl")
		if err != nil {
			// we don't care if this is missing, so we won't return an error
			slog.DebugContext(
				ctx,
				"missing imageUrl in Goodreads search result",
				slog.Any(otelkeys.Error, err),
			)
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

		if workID == 0 || bookID == 0 || title == "" {
			return
		}

		numPages, err := jsonparser.GetInt(value, "numPages")
		if err != nil {
			slog.DebugContext(ctx, "missing numPages in Goodreads search result", slog.Any(otelkeys.Error, err))
		}

		authorID, err := jsonparser.GetInt(value, "author", "id")
		if err != nil {
			slog.DebugContext(ctx, "missing author id in Goodreads search result", slog.Any(otelkeys.Error, err))
		}

		authorName, err := jsonparser.GetString(value, "author", "name")
		if err != nil {
			slog.DebugContext(ctx, "missing author name in Goodreads search result", slog.Any(otelkeys.Error, err))
		}

		results = append(results, BookResult{
			BookImageURL:      imageURL,
			BookLegacyID:      bookID,
			WorkLegacyID:      workID,
			BookTitle:         title,
			BookNumberOfPages: numPages,
			AuthorLegacyID:    authorID,
			AuthorName:        authorName,
		})
	})
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to parse Goodreads ISBN search response as JSON array",
			slog.Any(otelkeys.Error, err),
		)
	}

	return results
}
