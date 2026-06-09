package goodreads

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// GetBookByLegacyID fetches a single book from Goodreads using its numeric
// legacy work ID (the integer visible in older Goodreads URLs).
func (c *Client) GetBookByLegacyID(ctx context.Context, legacyID int64) (*BookResult, error) {
	resp, err := GetBookByLegacyId(ctx, c.client, legacyID)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to get book by legacy ID",
			slog.Int64(otelkeys.BookLegacyID, legacyID),
			slog.Any(otelkeys.Error, err),
		)
		return nil, fmt.Errorf("failed to get book by legacy ID: %w", err)
	}

	return loadBookResult(ctx, resp.GetBookByLegacyId.Work)
}

// GetBookByID fetches a single book from Goodreads using its GraphQL work ID
// (the opaque string used by the Goodreads GraphQL API).
func (c *Client) GetBookByID(ctx context.Context, grID string) (*BookResult, error) {
	resp, err := GetBook(ctx, c.client, grID)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to get book by Goodreads ID",
			slog.String(otelkeys.BookID, grID),
			slog.Any(otelkeys.Error, err),
		)
		return nil, fmt.Errorf("failed to get book by Goodreads ID: %w", err)
	}
	return loadBookResult(ctx, resp.GetBook.Work)
}

// GetBookByASIN fetches a single book from Goodreads using its Amazon ASIN.
func (c *Client) GetBookByASIN(ctx context.Context, asin string) (*BookResult, error) {
	resp, err := GetBookByAsin(ctx, c.client, asin)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to get book by ASIN",
			slog.String(otelkeys.BookASIN, asin),
			slog.Any(otelkeys.Error, err),
		)
		return nil, fmt.Errorf("failed to get book by ASIN: %w", err)
	}

	return loadBookResult(ctx, resp.GetBookByAsin.Work)
}

// workData holds common fields extracted from the various generated Work types.
// This avoids repeating the BookResult field mapping across each type branch.
type workData struct {
	workID, bookID, bookImageURL, bookTitle      string
	bookASIN, bookISBN, bookISBN13, bookLanguage string
	authorName, authorID, authorProfileImageURL  string
	workLegacyID, bookLegacyID, authorLegacyID   int64
}

type workNode interface {
	GetId() string
	GetLegacyId() int64
}

type bestBookNode interface {
	workNode
	GetImageUrl() string
	GetTitle() string
}

type bestBookDetailsNode interface {
	GetAsin() string
	GetIsbn() string
	GetIsbn13() string
}

type contributorNode interface {
	workNode
	GetName() string
	GetProfileImageUrl() string
}

func (d *workData) toBookResult() *BookResult {
	return &BookResult{
		WorkID:                d.workID,
		WorkLegacyID:          d.workLegacyID,
		BookID:                d.bookID,
		BookLegacyID:          d.bookLegacyID,
		BookImageURL:          d.bookImageURL,
		BookTitle:             d.bookTitle,
		BookASIN:              d.bookASIN,
		BookISBN:              d.bookISBN,
		BookISBN13:            d.bookISBN13,
		BookLanguage:          d.bookLanguage,
		AuthorName:            d.authorName,
		AuthorID:              d.authorID,
		AuthorLegacyID:        d.authorLegacyID,
		AuthorProfileImageURL: d.authorProfileImageURL,
	}
}

func extractWorkData(
	work workNode,
	book bestBookNode,
	details bestBookDetailsNode,
	language string,
	author contributorNode,
) *workData {
	return &workData{
		workID:                work.GetId(),
		workLegacyID:          work.GetLegacyId(),
		bookID:                book.GetId(),
		bookLegacyID:          book.GetLegacyId(),
		bookImageURL:          book.GetImageUrl(),
		bookTitle:             book.GetTitle(),
		bookASIN:              details.GetAsin(),
		bookISBN:              details.GetIsbn(),
		bookISBN13:            details.GetIsbn13(),
		bookLanguage:          language,
		authorName:            author.GetName(),
		authorID:              author.GetId(),
		authorLegacyID:        author.GetLegacyId(),
		authorProfileImageURL: author.GetProfileImageUrl(),
	}
}

// loadBookResult gets a BookResult from a Work type, which is the common type
// returned by all the different book queries. It handles all the different
// possible types of the Work field in the various queries.
func loadBookResult(ctx context.Context, work any) (*BookResult, error) {
	var d *workData

	switch v := work.(type) {
	case SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWork:
		book := &v.BestBook
		d = extractWorkData(
			&v,
			book,
			&book.Details,
			book.Details.Language.Name,
			&book.PrimaryContributorEdge.Node,
		)
	case GetBookByLegacyIdGetBookByLegacyIdBookWork:
		book := &v.BestBook
		d = extractWorkData(
			&v,
			book,
			&book.Details,
			book.Details.Language.Name,
			&book.PrimaryContributorEdge.Node,
		)
	case GetBookByAsinGetBookByAsinBookWork:
		book := &v.BestBook
		d = extractWorkData(
			&v,
			book,
			&book.Details,
			book.Details.Language.Name,
			&book.PrimaryContributorEdge.Node,
		)
	case GetBookGetBookWork:
		book := &v.BestBook
		d = extractWorkData(
			&v,
			book,
			&book.Details,
			book.Details.Language.Name,
			&book.PrimaryContributorEdge.Node,
		)
	default:
		slog.ErrorContext(
			ctx,
			"unexpected type for book result",
			slog.String(otelkeys.Type, fmt.Sprintf("%T", work)),
		)
		return nil, fmt.Errorf("unexpected type for book result: %T", work)
	}

	return d.toBookResult(), nil
}
