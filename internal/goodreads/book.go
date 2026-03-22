package goodreads

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

func (c *Client) GetBookByID(ctx context.Context, grID string) (*BookResult, error) {
	resp, err := GetBook(ctx, c.client, grID)
	if err != nil {
		return nil, err
	}
	return loadBookResult(ctx, resp.GetBook.Work.BestBook)
}

func (c *Client) GetBookByASIN(ctx context.Context, asin string) (*BookResult, error) {
	resp, error := GetBookByAsin(ctx, c.client, asin)
	if error != nil {
		return nil, error
	}

	return loadBookResult(ctx, resp.GetBookByAsin.Work.BestBook)
}

func loadBookResult(ctx context.Context, value any) (result *BookResult, err error) {
	// see if this is a GetBookByLegacyIdGetBookByLegacyIdBookWork or SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWork
	if v, ok := value.(GetBookByLegacyIdGetBookByLegacyIdBookWork); ok {
		book := v.BestBook
		author := book.PrimaryContributorEdge.Node
		return &BookResult{
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
		}, nil
	}
	if v, ok := value.(SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWork); ok {
		book := v.BestBook
		author := book.PrimaryContributorEdge.Node
		return &BookResult{
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
		}, nil
	}
	if v, ok := value.(GetBookByLegacyIdGetBookByLegacyIdBookWorkBestBook); ok {
		book := v
		author := book.PrimaryContributorEdge.Node
		return &BookResult{
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
		}, nil
	}
	if v, ok := value.(GetBookByAsinGetBookByAsinBookWorkBestBook); ok {
		book := v
		author := book.PrimaryContributorEdge.Node
		return &BookResult{
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
		}, nil
	}
	if v, ok := value.(GetBookGetBookWorkBestBook); ok {
		book := v
		author := book.PrimaryContributorEdge.Node
		return &BookResult{
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
		}, nil
	}
	slog.ErrorContext(ctx, "unexpected type for book result", slog.Any(otelkeys.Type, value))
	return nil, fmt.Errorf("unexpected type for book result: %T", value)
}
