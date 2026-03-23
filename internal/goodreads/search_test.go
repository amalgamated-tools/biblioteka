package goodreads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/require"
)

// mockGraphQLClient implements graphql.Client for testing the Search method.
type mockGraphQLClient struct {
	handler func(req *graphql.Request, resp *graphql.Response) error
}

func (m *mockGraphQLClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	return m.handler(req, resp)
}

type mockHTTPClient struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

func noResponseClient() *Client {
	return &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				resp.Data = nil
				// to not append to the results slice in case of missing required fields,
				// we return an error here which will cause the parseISBNSearchResponse to skip processing this entry
				return errors.New("missing required fields in response")
			},
		},
	}
}

func TestSearch_Success(t *testing.T) {
	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				data := resp.Data.(*SearchResponse)
				data.GetSearchSuggestions = SearchGetSearchSuggestionsSearchResultsConnection{
					TotalCount: 1,
					Edges: []SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchResultEdge{
						&SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdge{
							Typename: "SearchBookEdge",
							Node: SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBook{
								Work: SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdgeNodeBookWork{
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
											NumPages: 476,
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
								},
							},
						},
					},
				}
				return nil
			},
		},
	}

	results, err := client.Search(context.Background(), "project hail mary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.WorkID != "kca://work/amzn1.gr.work.v1.abc123" {
		t.Errorf("WorkID = %q, want %q", r.WorkID, "kca://work/amzn1.gr.work.v1.abc123")
	}
	if r.WorkLegacyID != 79106958 {
		t.Errorf("WorkLegacyID = %d, want %d", r.WorkLegacyID, 79106958)
	}
	if r.BookID != "kca://book/amzn1.gr.book.v1.def456" {
		t.Errorf("BookID = %q, want %q", r.BookID, "kca://book/amzn1.gr.book.v1.def456")
	}
	if r.BookLegacyID != 54493401 {
		t.Errorf("BookLegacyID = %d, want %d", r.BookLegacyID, 54493401)
	}
	if r.BookImageURL != "https://example.com/image.jpg" {
		t.Errorf("BookImageURL = %q, want %q", r.BookImageURL, "https://example.com/image.jpg")
	}
	if r.BookTitle != "Project Hail Mary" {
		t.Errorf("BookTitle = %q, want %q", r.BookTitle, "Project Hail Mary")
	}
	if r.BookASIN != "B08FHBV4ZX" {
		t.Errorf("BookASIN = %q, want %q", r.BookASIN, "B08FHBV4ZX")
	}
	if r.BookISBN != "0593135202" {
		t.Errorf("BookISBN = %q, want %q", r.BookISBN, "0593135202")
	}
	if r.BookISBN13 != "9780593135204" {
		t.Errorf("BookISBN13 = %q, want %q", r.BookISBN13, "9780593135204")
	}
	if r.BookLanguage != "English" {
		t.Errorf("BookLanguage = %q, want %q", r.BookLanguage, "English")
	}
	if r.BookNumberOfPages != 476 {
		t.Errorf("BookNumberOfPages = %d, want %d", r.BookNumberOfPages, 476)
	}
	if r.AuthorName != "Andy Weir" {
		t.Errorf("AuthorName = %q, want %q", r.AuthorName, "Andy Weir")
	}
	if r.AuthorID != "kca://author/amzn1.gr.author.v1.ghi789" {
		t.Errorf("AuthorID = %q, want %q", r.AuthorID, "kca://author/amzn1.gr.author.v1.ghi789")
	}
	if r.AuthorLegacyID != 6540057 {
		t.Errorf("AuthorLegacyID = %d, want %d", r.AuthorLegacyID, 6540057)
	}
	if r.AuthorProfileImageURL != "https://example.com/author.jpg" {
		t.Errorf("AuthorProfileImageURL = %q, want %q", r.AuthorProfileImageURL, "https://example.com/author.jpg")
	}
}

func TestSearch_GraphQLError(t *testing.T) {
	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				return fmt.Errorf("graphql request failed")
			},
		},
	}

	results, err := client.Search(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_EmptyEdges(t *testing.T) {
	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				data := resp.Data.(*SearchResponse)
				data.GetSearchSuggestions = SearchGetSearchSuggestionsSearchResultsConnection{
					TotalCount: 0,
					Edges:      nil,
				}
				return nil
			},
		},
	}

	results, err := client.Search(context.Background(), "nonexistent book xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func Test_SearchByISBN(t *testing.T) {
	var response GetBookByLegacyIdResponse
	err := json.Unmarshal(GetBookByLegacyID_54493401, &response)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON response: %v", err)
	}
	client := &Client{
		httpClient: &mockHTTPClient{
			handler: func(req *http.Request) (*http.Response, error) {
				// Return a successful response with the JSON body
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(AutoComplete)),
					Header:     make(http.Header),
				}, nil
			},
		},
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				data := resp.Data.(*GetBookByLegacyIdResponse)
				*data = response
				return nil
			},
		},
	}

	results, err := client.SearchByISBN(t.Context(), response.GetBookByLegacyId.Work.BestBook.Details.Isbn)
	if err != nil {
		t.Fatalf("failed to search by ISBN: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	expected, err := loadBookResult(t.Context(), response.GetBookByLegacyId.Work)
	if err != nil {
		t.Fatalf("failed to load expected BookResult from response: %v", err)
	}

	require.Equal(t, *expected, r)
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
	require.Equal(t, expected.BookNumberOfPages, r.BookNumberOfPages)
	require.Equal(t, expected.AuthorID, r.AuthorID)
	require.Equal(t, expected.AuthorName, r.AuthorName)
	require.Equal(t, expected.AuthorLegacyID, r.AuthorLegacyID)
	require.Equal(t, expected.AuthorProfileImageURL, r.AuthorProfileImageURL)
}

func TestParseISBNSearchResponse_Success(t *testing.T) {
	var response GetBookByLegacyIdResponse
	err := json.Unmarshal(GetBookByLegacyID_54493401, &response)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON response: %v", err)
	}
	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				data := resp.Data.(*GetBookByLegacyIdResponse)
				*data = response
				return nil
			},
		},
	}

	results := client.parseISBNSearchResponse(context.Background(), AutoComplete)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	expected, err := loadBookResult(t.Context(), response.GetBookByLegacyId.Work)
	if err != nil {
		t.Fatalf("failed to load expected BookResult from response: %v", err)
	}

	require.Equal(t, *expected, r)
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
	require.Equal(t, expected.BookNumberOfPages, r.BookNumberOfPages)
	require.Equal(t, expected.AuthorID, r.AuthorID)
	require.Equal(t, expected.AuthorName, r.AuthorName)
	require.Equal(t, expected.AuthorLegacyID, r.AuthorLegacyID)
	require.Equal(t, expected.AuthorProfileImageURL, r.AuthorProfileImageURL)
}

func TestParseISBNSearchResponse_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing bookId",
			body: `[{"workId": "123", "title": "Test"}]`,
		},
		{
			name: "missing workId",
			body: `[{"bookId": "123", "title": "Test"}]`,
		},
		{
			name: "missing title",
			body: `[{"bookId": "123", "workId": "456"}]`,
		},
		{
			name: "zero workId",
			body: `[{"bookId": "123", "workId": "0", "title": "Test"}]`,
		},
		{
			name: "zero bookId",
			body: `[{"bookId": "0", "workId": "456", "title": "Test"}]`,
		},
		{
			name: "empty title",
			body: `[{"bookId": "123", "workId": "456", "title": ""}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				client: &mockGraphQLClient{
					handler: func(req *graphql.Request, resp *graphql.Response) error {
						resp.Data = nil
						// to not append to the results slice in case of missing required fields,
						// we return an error here which will cause the parseISBNSearchResponse to skip processing this entry
						return errors.New("missing required fields in response")
					},
				},
			}
			results := client.parseISBNSearchResponse(context.Background(), []byte(tt.body))
			if len(results) != 0 {
				t.Errorf("expected 0 results, got %d", len(results))
			}
		})
	}
}

func TestParseISBNSearchResponse_InvalidBookID(t *testing.T) {
	body := `[{"bookId": "not-a-number", "workId": "456", "title": "Test"}]`
	results := noResponseClient().parseISBNSearchResponse(context.Background(), []byte(body))
	if len(results) != 0 {
		t.Errorf("expected 0 results for invalid bookId, got %d", len(results))
	}
}

func TestParseISBNSearchResponse_InvalidWorkID(t *testing.T) {
	body := `[{"bookId": "123", "workId": "not-a-number", "title": "Test"}]`
	results := noResponseClient().parseISBNSearchResponse(context.Background(), []byte(body))
	if len(results) != 0 {
		t.Errorf("expected 0 results for invalid workId, got %d", len(results))
	}
}

func TestParseISBNSearchResponse_OptionalFieldsMissing(t *testing.T) {
	body := `[{"bookId": "123", "workId": "456", "title": "Minimal Book"}]`
	results := noResponseClient().parseISBNSearchResponse(context.Background(), []byte(body))
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.BookTitle != "Minimal Book" {
		t.Errorf("BookTitle = %q, want %q", r.BookTitle, "Minimal Book")
	}
	if r.BookImageURL != "" {
		t.Errorf("BookImageURL = %q, want empty", r.BookImageURL)
	}
	if r.BookNumberOfPages != 0 {
		t.Errorf("BookNumberOfPages = %d, want 0", r.BookNumberOfPages)
	}
	if r.AuthorName != "" {
		t.Errorf("AuthorName = %q, want empty", r.AuthorName)
	}
}

func TestParseISBNSearchResponse_EmptyArray(t *testing.T) {
	results := noResponseClient().parseISBNSearchResponse(context.Background(), []byte(`[]`))
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseISBNSearchResponse_InvalidJSON(t *testing.T) {
	results := noResponseClient().parseISBNSearchResponse(context.Background(), []byte(`not json`))
	if len(results) != 0 {
		t.Errorf("expected 0 results for invalid JSON, got %d", len(results))
	}
}

func TestParseISBNSearchResponse_MultipleResults(t *testing.T) {
	body := `[
		{"bookId": "111", "workId": "222", "title": "Book One", "numPages": 100, "author": {"id": 1, "name": "Author A"}},
		{"bookId": "333", "workId": "444", "title": "Book Two", "numPages": 200, "author": {"id": 2, "name": "Author B"}}
	]`
	results := noResponseClient().parseISBNSearchResponse(context.Background(), []byte(body))
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].BookTitle != "Book One" {
		t.Errorf("results[0].BookTitle = %q, want %q", results[0].BookTitle, "Book One")
	}
	if results[0].BookLegacyID != 111 {
		t.Errorf("results[0].BookLegacyID = %d, want %d", results[0].BookLegacyID, 111)
	}
	if results[1].BookTitle != "Book Two" {
		t.Errorf("results[1].BookTitle = %q, want %q", results[1].BookTitle, "Book Two")
	}
	if results[1].AuthorName != "Author B" {
		t.Errorf("results[1].AuthorName = %q, want %q", results[1].AuthorName, "Author B")
	}
}

func TestParseISBNSearchResponse_SkipsInvalidEntriesKeepsValid(t *testing.T) {
	body := `[
		{"bookId": "bad", "workId": "222", "title": "Invalid Entry"},
		{"bookId": "333", "workId": "444", "title": "Valid Entry", "author": {"id": 1, "name": "Author"}}
	]`
	results := noResponseClient().parseISBNSearchResponse(context.Background(), []byte(body))
	if len(results) != 1 {
		t.Fatalf("expected 1 result (skipping invalid), got %d", len(results))
	}
	if results[0].BookTitle != "Valid Entry" {
		t.Errorf("BookTitle = %q, want %q", results[0].BookTitle, "Valid Entry")
	}
}
