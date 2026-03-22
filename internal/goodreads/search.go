package goodreads

import (
	"context"
	"fmt"
	"io"
	"net/http"

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
			Title:                 res.Work.GetBestBook().Title,
			AuthorName:            res.Work.GetBestBook().PrimaryContributorEdge.Node.Name,
			AuthorID:              res.Work.GetBestBook().PrimaryContributorEdge.Node.Id,
			AuthorLegacyID:        res.Work.GetBestBook().PrimaryContributorEdge.Node.LegacyId,
			AuthorProfileImageURL: res.Work.GetBestBook().PrimaryContributorEdge.Node.ProfileImageUrl,
		})
	}

	return results, nil
}

func (c *Client) SearchByISBN(ctx context.Context, isbn string) ([]SearchResult, error) {
	results := make([]SearchResult, 0)
	url := fmt.Sprintf("https://goodreads.com/book/auto_complete?format=json&q=%s", isbn)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return results, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return results, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return results, fmt.Errorf("failed to read response body: %w", err)
	}

	jsonparser.ArrayEach(bodyText, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err != nil {
			return
		}

		workID, err := jsonparser.GetInt(value, "workId")
		if err != nil {
			return
		}
		bookID, err := jsonparser.GetInt(value, "bookId")
		if err != nil {
			return
		}
		if workID == 0 || bookID == 0 {
			return
		}
		// title, _ := jsonparser.GetString(value, "title")
		// author, _ := jsonparser.GetString(value, "author_name")

		results = append(results, SearchResult{
			WorkLegacyID: workID,
			BookLegacyID: bookID,
			// WorkLegacyID:   res.Work.LegacyId,
			// BookID:         res.Work.GetBestBook().Id,
			// BookLegacyID:   res.Work.GetBestBook().LegacyId,
			// Title:          res.Work.GetBestBook().Title,
			// Author:         res.Work.GetBestBook().PrimaryContributorEdge.Node.Name,
			// AuthorID:       res.Work.GetBestBook().PrimaryContributorEdge.Node.Id,
			// AuthorLegacyID: res.Work.GetBestBook().PrimaryContributorEdge.Node.LegacyId,
		})
	})

	return results, nil
}
