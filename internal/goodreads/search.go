package goodreads

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// Search performs a search query against the Goodreads unpublished GraphQL API and returns a list of search results.
// The query parameter is a string that can contain the book title, author name, or other relevant information to find matching books on Goodreads.
// It should not be a search for ISBN - we have other functions for that
func (c *Client) Search(ctx context.Context, query string) ([]BookResult, error) {
	results := make([]BookResult, 0)
	resp, err := Search(ctx, c.client, query)
	if err != nil {
		return results, fmt.Errorf("goodreads search: %w", err)
	}

	for _, e := range resp.GetSearchSuggestions.Edges {
		edge, ok := e.(*SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdge)
		if !ok {
			slog.DebugContext(
				ctx,
				"unexpected edge type in Goodreads search results",
				slog.Any(otelkeys.EdgeType, fmt.Sprintf("%T", e)),
			)
			continue
		}
		res := edge.Node

		result, err := loadBookResult(ctx, res.Work)
		if err != nil {
			slog.DebugContext(
				ctx,
				"failed to load book result from Goodreads search result",
				slog.Any(otelkeys.Error, err),
				slog.Any(otelkeys.WorkID, res.Work.Id),
			)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}
