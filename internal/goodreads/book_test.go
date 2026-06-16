package goodreads

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/require"
)

func Test_loadBookResultSupportedWorkTypes(t *testing.T) {
	var byASIN GetBookByAsinResponse
	err := json.Unmarshal(GetBookByAsin_B08FHBV4ZX, &byASIN)
	require.NoError(t, err)

	var byID GetBookResponse
	err = json.Unmarshal(GetBook_7WmufEffpivF1XTp, &byID)
	require.NoError(t, err)

	var byLegacyID GetBookByLegacyIdResponse
	err = json.Unmarshal(GetBookByLegacyID_54493401, &byLegacyID)
	require.NoError(t, err)

	searchWork := SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWork{
		Id:       "kca://work/amzn1.gr.work.v1.abc123",
		LegacyId: 79106958,
		BestBook: SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWorkBestBook{
			Id:       "kca://book/amzn1.gr.book.v1.def456",
			LegacyId: 54493401,
			ImageUrl: "https://example.com/image.jpg",
			Title:    "Project Hail Mary",
			Details: SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWorkBestBookDetails{
				Asin:   "B08FHBV4ZX",
				Isbn:   "0593135202",
				Isbn13: "9780593135204",
				Language: SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWorkBestBookDetailsLanguage{
					Name: "English",
				},
			},
			PrimaryContributorEdge: SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWorkBestBookPrimaryContributorEdgeBookContributorEdge{
				Node: SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWorkBestBookPrimaryContributorEdgeBookContributorEdgeNodeContributor{
					Id:              "kca://author/amzn1.gr.author.v1.ghi789",
					Name:            "Andy Weir",
					LegacyId:        6540057,
					ProfileImageUrl: "https://example.com/author.jpg",
				},
			},
		},
	}

	testCases := []struct {
		name string
		work any
	}{
		{
			name: "search",
			work: searchWork,
		},
		{
			name: "legacy id",
			work: byLegacyID.GetBookByLegacyId.Work,
		},
		{
			name: "asin",
			work: byASIN.GetBookByAsin.Work,
		},
		{
			name: "id",
			work: byID.GetBook.Work,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := loadBookResult(t.Context(), tc.work)
			require.NoError(t, err)

			var (
				expWorkID    string
				expBookID    string
				expAuthorID  string
				expBookTitle string
				expBookASIN  string
			)

			switch w := tc.work.(type) {
			case SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWork:
				expWorkID = w.Id
				expBookID = w.BestBook.Id
				expAuthorID = w.BestBook.PrimaryContributorEdge.Node.Id
				expBookTitle = w.BestBook.Title
				expBookASIN = w.BestBook.Details.Asin
			case GetBookByLegacyIdGetBookByLegacyIdBookWork:
				expWorkID = w.Id
				expBookID = w.BestBook.Id
				expAuthorID = w.BestBook.PrimaryContributorEdge.Node.Id
				expBookTitle = w.BestBook.Title
				expBookASIN = w.BestBook.Details.Asin
			case GetBookByAsinGetBookByAsinBookWork:
				expWorkID = w.Id
				expBookID = w.BestBook.Id
				expAuthorID = w.BestBook.PrimaryContributorEdge.Node.Id
				expBookTitle = w.BestBook.Title
				expBookASIN = w.BestBook.Details.Asin
			case GetBookGetBookWork:
				expWorkID = w.Id
				expBookID = w.BestBook.Id
				expAuthorID = w.BestBook.PrimaryContributorEdge.Node.Id
				expBookTitle = w.BestBook.Title
				expBookASIN = w.BestBook.Details.Asin
			default:
				require.FailNow(t, "unexpected work type", "%T", tc.work)
			}

			require.Equal(t, expWorkID, result.WorkID)
			require.Equal(t, expBookID, result.BookID)
			require.Equal(t, expAuthorID, result.AuthorID)
			require.Equal(t, expBookTitle, result.BookTitle)
			require.Equal(t, expBookASIN, result.BookASIN)
		})
	}
}

func Test_GetBookByASIN(t *testing.T) {
	var response GetBookByAsinResponse
	err := json.Unmarshal(GetBookByAsin_B08FHBV4ZX, &response)
	require.NoError(t, err, "failed to unmarshal JSON response")
	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				data := resp.Data.(*GetBookByAsinResponse)
				*data = response
				return nil
			},
		},
	}

	r, err := client.GetBookByASIN(t.Context(), response.GetBookByAsin.Work.BestBook.Details.Asin)
	require.NoError(t, err, "failed to search by ASIN")

	expected, err := loadBookResult(t.Context(), response.GetBookByAsin.Work)
	require.NoError(t, err, "failed to load expected BookResult from response")

	require.Equal(t, expected, r)
	require.Equal(t, expected.WorkID, r.WorkID)
	require.Equal(t, expected.WorkLegacyID, r.WorkLegacyID)
	require.Equal(t, expected.BookID, r.BookID)
	require.Equal(t, expected.BookLegacyID, r.BookLegacyID)
	require.Equal(t, expected.BookImageURL, r.BookImageURL)
	require.Equal(t, expected.BookTitle, r.BookTitle)
	require.Equal(t, expected.BookASIN, r.BookASIN)
	require.Equal(t, expected.BookISBN, r.BookISBN)
	require.Equal(t, expected.BookISBN13, r.BookISBN13)
	require.Equal(t, expected.BookLanguage, r.BookLanguage)
	require.Equal(t, expected.AuthorID, r.AuthorID)
	require.Equal(t, expected.AuthorName, r.AuthorName)
	require.Equal(t, expected.AuthorLegacyID, r.AuthorLegacyID)
	require.Equal(t, expected.AuthorProfileImageURL, r.AuthorProfileImageURL)
}

func Test_GetBookByASINError(t *testing.T) {
	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				return errors.New("mock error")
			},
		},
	}

	_, err := client.GetBookByASIN(t.Context(), "mock-asin")
	require.Error(t, err, "expected error, got nil")
}

func Test_GetBook(t *testing.T) {
	var response GetBookResponse
	err := json.Unmarshal(GetBook_7WmufEffpivF1XTp, &response)
	require.NoError(t, err, "failed to unmarshal JSON response")
	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				data := resp.Data.(*GetBookResponse)
				*data = response
				return nil
			},
		},
	}

	r, err := client.GetBookByID(t.Context(), response.GetBook.Work.BestBook.Id)
	require.NoError(t, err, "failed to search by ID")

	expected, err := loadBookResult(t.Context(), response.GetBook.Work)
	require.NoError(t, err, "failed to load expected BookResult from response")

	require.Equal(t, expected, r)
	require.Equal(t, expected.WorkID, r.WorkID)
	require.Equal(t, expected.WorkLegacyID, r.WorkLegacyID)
	require.Equal(t, expected.BookID, r.BookID)
	require.Equal(t, expected.BookLegacyID, r.BookLegacyID)
	require.Equal(t, expected.BookImageURL, r.BookImageURL)
	require.Equal(t, expected.BookTitle, r.BookTitle)
	require.Equal(t, expected.BookASIN, r.BookASIN)
	require.Equal(t, expected.BookISBN, r.BookISBN)
	require.Equal(t, expected.BookISBN13, r.BookISBN13)
	require.Equal(t, expected.BookLanguage, r.BookLanguage)
	require.Equal(t, expected.AuthorID, r.AuthorID)
	require.Equal(t, expected.AuthorName, r.AuthorName)
	require.Equal(t, expected.AuthorLegacyID, r.AuthorLegacyID)
	require.Equal(t, expected.AuthorProfileImageURL, r.AuthorProfileImageURL)
}
