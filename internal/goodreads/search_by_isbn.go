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
	// these can be 10 or 13 characters long, and can contain dashes, but we will just remove the dashes and check the length of the remaining string
	var isbnDigits strings.Builder
	for _, r := range isbn {
		if r >= '0' && r <= '9' {
			isbnDigits.WriteString(string(r))
		}
	}
	if len(isbnDigits.String()) != 10 && len(isbnDigits.String()) != 13 {
		slog.ErrorContext(
			ctx,
			"invalid ISBN length",
			slog.Any(otelkeys.ISBN, isbn),
		)
		return nil, fmt.Errorf("invalid ISBN: %s", isbn)
	}
	if len(isbnDigits.String()) == 10 && !ValidISBN10CheckDigit(isbn) {
		slog.ErrorContext(
			ctx,
			"invalid ISBN-10 check digit",
			slog.Any(otelkeys.ISBN, isbn),
		)
		return nil, fmt.Errorf("invalid ISBN-10 check digit: %s", isbn)
	}
	if len(isbnDigits.String()) == 13 && !ValidISBN13CheckDigit(isbn) {
		slog.ErrorContext(
			ctx,
			"invalid ISBN-13 check digit",
			slog.Any(otelkeys.ISBN, isbn),
		)
		return nil, fmt.Errorf("invalid ISBN-13 check digit: %s", isbn)
	}

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
		slog.ErrorContext(
			ctx,
			"goodreads ISBN search returned non-OK status",
			slog.Any(otelkeys.Query, isbn),
			slog.Any(otelkeys.StatusCode, resp.StatusCode),
			slog.Any(otelkeys.URL, searchURL),
		)
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
			slog.ErrorContext(
				ctx,
				"failed to get bookId from Goodreads ISBN search result",
				slog.Any(otelkeys.Error, err),
			)
			return
		}

		bookID, err := strconv.ParseInt(bookIDStr, 10, 64)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"failed to parse bookId from Goodreads ISBN search result",
				slog.Any(otelkeys.Error, err),
			)
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
			"failed to get book by legacy ID, returning partial result",
			slog.Int64(otelkeys.BookLegacyID, bookID),
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
			slog.ErrorContext(
				ctx,
				"failed to get workId from Goodreads ISBN search result",
				slog.Any(otelkeys.Error, err),
			)
			return
		}

		workID, err := strconv.ParseInt(workIDStr, 10, 64)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"failed to parse workId from Goodreads ISBN search result",
				slog.Any(otelkeys.Error, err),
			)
			return
		}

		title, err := jsonparser.GetString(value, "title")
		if err != nil {
			slog.ErrorContext(
				ctx,
				"failed to get title from Goodreads ISBN search result",
				slog.Any(otelkeys.Error, err),
			)
			return
		}

		if workID == 0 || bookID == 0 || title == "" {
			slog.ErrorContext(
				ctx,
				"missing required fields in Goodreads ISBN search result",
				slog.Int64(otelkeys.BookLegacyID, bookID),
				slog.Int64(otelkeys.WorkLegacyID, workID),
				slog.Any(otelkeys.Title, title),
			)
			return
		}

		numPages, err := jsonparser.GetInt(value, "numPages")
		if err != nil {
			// we don't care if this is missing, so we won't return an error
			slog.DebugContext(ctx, "missing numPages in Goodreads search result", slog.Any(otelkeys.Error, err))
		}

		authorID, err := jsonparser.GetInt(value, "author", "id")
		if err != nil {
			// this is a problem
			slog.ErrorContext(
				ctx,
				"failed to get author ID from Goodreads ISBN search result",
				slog.Any(otelkeys.Error, err),
			)
			return
		}

		authorName, err := jsonparser.GetString(value, "author", "name")
		if err != nil {
			// we don't care if this is missing, so we won't return an error
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

	slog.DebugContext(
		ctx,
		"parsed Goodreads ISBN search response",
		slog.Int(otelkeys.Count, len(results)),
	)
	return results
}
