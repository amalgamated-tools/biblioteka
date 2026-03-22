package goodreads

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

func (c *Client) GetBookByLegacyID(ctx context.Context, legacyID int64) (*BookResult, error) {
	resp, err := GetBookByLegacyId(ctx, c.client, legacyID)
	if err != nil {
		return nil, err
	}

	return loadBookResult(ctx, resp.GetBookByLegacyId.Work)
}

func (c *Client) GetBookByID(ctx context.Context, grID string) (*BookResult, error) {
	resp, err := GetBook(ctx, c.client, grID)
	if err != nil {
		return nil, err
	}
	return loadBookResult(ctx, resp.GetBook.Work)
}

func (c *Client) GetBookByASIN(ctx context.Context, asin string) (*BookResult, error) {
	resp, error := GetBookByAsin(ctx, c.client, asin)
	if error != nil {
		return nil, error
	}

	return loadBookResult(ctx, resp.GetBookByAsin.Work)
}

// loadBookResult gets a bookresult from a *Work type, which is the common type returned by all the different book queries.
// It handles all the different possible types of the Work field in the various queries.
func loadBookResult(ctx context.Context, work any) (*BookResult, error) {
	var result *BookResult

	// we reach this from the goodreadsClient.Search method, which hits their graphql endpoint,
	// and the type of the work field is SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWork
	if v, ok := work.(SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWork); ok {
		book := v.BestBook
		author := book.PrimaryContributorEdge.Node
		result = &BookResult{
			WorkID:                v.Id,
			WorkLegacyID:          v.LegacyId,
			BookID:                book.Id,
			BookLegacyID:          book.LegacyId,
			BookImageURL:          book.ImageUrl,
			BookTitle:             book.Title,
			BookASIN:              book.Details.Asin,
			BookISBN:              book.Details.Isbn,
			BookISBN13:            book.Details.Isbn13,
			BookLanguage:          book.Details.Language.Name,
			BookNumberOfPages:     book.Details.NumPages,
			AuthorName:            author.Name,
			AuthorID:              author.Id,
			AuthorLegacyID:        author.LegacyId,
			AuthorProfileImageURL: author.ProfileImageUrl,
		}
	}

	// we reach this from the goodreadsClient.GetBookByLegacyId method, which hits their graphql endpoint
	// sometimes after hitting the autocomplete (when searching by ISBN)
	if v, ok := work.(GetBookByLegacyIdGetBookByLegacyIdBookWork); ok {
		book := v.BestBook
		author := book.PrimaryContributorEdge.Node
		result = &BookResult{
			WorkID:                v.Id,
			WorkLegacyID:          v.LegacyId,
			BookID:                book.Id,
			BookLegacyID:          book.LegacyId,
			BookImageURL:          book.ImageUrl,
			BookTitle:             book.Title,
			BookASIN:              book.Details.Asin,
			BookISBN:              book.Details.Isbn,
			BookISBN13:            book.Details.Isbn13,
			BookLanguage:          book.Details.Language.Name,
			BookNumberOfPages:     book.Details.NumPages,
			AuthorName:            author.Name,
			AuthorID:              author.Id,
			AuthorLegacyID:        author.LegacyId,
			AuthorProfileImageURL: author.ProfileImageUrl,
		}
	}

	// we reach this from the goodreadsClient.GetBookByASIN method, which hits their graphql endpoint
	if v, ok := work.(GetBookByAsinGetBookByAsinBookWork); ok {
		book := v.BestBook
		author := book.PrimaryContributorEdge.Node
		result = &BookResult{
			WorkID:                v.Id,
			WorkLegacyID:          v.LegacyId,
			BookID:                book.Id,
			BookLegacyID:          book.LegacyId,
			BookImageURL:          book.ImageUrl,
			BookTitle:             book.Title,
			BookASIN:              book.Details.Asin,
			BookISBN:              book.Details.Isbn,
			BookISBN13:            book.Details.Isbn13,
			BookLanguage:          book.Details.Language.Name,
			BookNumberOfPages:     book.Details.NumPages,
			AuthorName:            author.Name,
			AuthorID:              author.Id,
			AuthorLegacyID:        author.LegacyId,
			AuthorProfileImageURL: author.ProfileImageUrl,
		}
	}

	// we reach this from the goodreadsClient.GetBook method, which hits their graphql endpoint
	if v, ok := work.(GetBookGetBookWork); ok {
		book := v.BestBook
		author := book.PrimaryContributorEdge.Node
		result = &BookResult{
			WorkID:                v.Id,
			WorkLegacyID:          v.LegacyId,
			BookID:                book.Id,
			BookLegacyID:          book.LegacyId,
			BookImageURL:          book.ImageUrl,
			BookTitle:             book.Title,
			BookASIN:              book.Details.Asin,
			BookISBN:              book.Details.Isbn,
			BookISBN13:            book.Details.Isbn13,
			BookLanguage:          book.Details.Language.Name,
			BookNumberOfPages:     book.Details.NumPages,
			AuthorName:            author.Name,
			AuthorID:              author.Id,
			AuthorLegacyID:        author.LegacyId,
			AuthorProfileImageURL: author.ProfileImageUrl,
		}
	}
	if result != nil {
		return result, nil
	}
	slog.ErrorContext(ctx, "unexpected type for book result", slog.Any(otelkeys.Type, work))
	return result, fmt.Errorf("unexpected type for book result: %T", work)
}
