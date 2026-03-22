package goodreads

import "context"

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
			WorkID:       res.GetWork().Id,
			WorkLegacyID: (res.GetWork().LegacyId),
		})
	}

	return results, nil
}
