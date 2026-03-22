package goodreads

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/buger/jsonparser"
)

// Search performs a search query against the Goodreads unpublished GraphQL API and returns a list of search results.
// The query parameter is a string that can contain the book title, author name, or other relevant information to find matching books on Goodreads.
// It should not be a search for ISBN - we have other functions for that
func (c *Client) Search(ctx context.Context, query string) ([]SearchResult, error) {
	results := make([]SearchResult, 0)
	resp, err := Search(ctx, c.client, query)
	if err != nil {
		return results, err
	}

	for _, e := range resp.GetSearchSuggestions.Edges {
		edge, ok := e.(*SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdge)
		if !ok {
			continue
		}
		res := edge.Node
		results = append(results, SearchResult{
			WorkID:                res.Work.Id,
			WorkLegacyID:          res.Work.LegacyId,
			BookID:                res.Work.GetBestBook().Id,
			BookLegacyID:          res.Work.GetBestBook().LegacyId,
			BookImageURL:          res.Work.GetBestBook().ImageUrl,
			BookTitle:             res.Work.GetBestBook().Title,
			BookASIN:              res.Work.GetBestBook().Details.Asin,
			BookISBN:              res.Work.GetBestBook().Details.Isbn,
			BookISBN13:            res.Work.GetBestBook().Details.Isbn13,
			BookLanguage:          res.Work.GetBestBook().Details.Language.Name,
			BookNumberOfPages:     res.Work.GetBestBook().Details.NumPages,
			AuthorName:            res.Work.GetBestBook().PrimaryContributorEdge.Node.Name,
			AuthorID:              res.Work.GetBestBook().PrimaryContributorEdge.Node.Id,
			AuthorLegacyID:        res.Work.GetBestBook().PrimaryContributorEdge.Node.LegacyId,
			AuthorProfileImageURL: res.Work.GetBestBook().PrimaryContributorEdge.Node.ProfileImageUrl,
		})
	}

	return results, nil
}

func (c *Client) SearchByISBN(ctx context.Context, isbn string) ([]SearchResult, error) {
	url := fmt.Sprintf("https://goodreads.com/book/auto_complete?format=json&q=%s", isbn)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return parseISBNSearchResponse(ctx, bodyText), nil
}

// parseISBNSearchResponse parses the JSON response from the Goodreads auto_complete endpoint.
func parseISBNSearchResponse(ctx context.Context, bodyText []byte) []SearchResult {
	results := make([]SearchResult, 0)

	_, _ = jsonparser.ArrayEach(bodyText, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err != nil {
			return
		}

		imageURL, err := jsonparser.GetString(value, "imageUrl")
		if err != nil {
			// we don't care if this is missing, so we won't return an error
			slog.DebugContext(ctx, "missing imageUrl in Goodreads search result", slog.Any(otelkeys.Error, err))
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

		results = append(results, SearchResult{
			BookImageURL:      imageURL,
			BookLegacyID:      bookID,
			WorkLegacyID:      workID,
			BookTitle:         title,
			BookNumberOfPages: numPages,
			AuthorLegacyID:    authorID,
			AuthorName:        authorName,
		})
	})

	return results
}
