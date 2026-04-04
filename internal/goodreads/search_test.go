package goodreads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
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

	results, err := client.Search(t.Context(), "project hail mary")
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

	results, err := client.Search(t.Context(), "test")
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

	results, err := client.Search(t.Context(), "nonexistent book xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_DeduplicatesBooks(t *testing.T) {
	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				data := resp.Data.(*SearchResponse)
				duplicate := &SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchBookEdge{
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
				}
				data.GetSearchSuggestions = SearchGetSearchSuggestionsSearchResultsConnection{
					TotalCount: 2,
					Edges: []SearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchResultEdge{
						duplicate,
						duplicate,
					},
				}
				return nil
			},
		},
	}

	results, err := client.Search(t.Context(), "project hail mary")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "kca://book/amzn1.gr.book.v1.def456", results[0].BookID)
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
	require.Equal(t, expected.AuthorID, r.AuthorID)
	require.Equal(t, expected.AuthorName, r.AuthorName)
	require.Equal(t, expected.AuthorLegacyID, r.AuthorLegacyID)
	require.Equal(t, expected.AuthorProfileImageURL, r.AuthorProfileImageURL)
}

func Test_SearchByISBN_NormalizesQuery(t *testing.T) {
	tests := []struct {
		name      string
		isbn      string
		wantQuery string
	}{
		{
			name:      "strips ISBN-13 separators",
			isbn:      "978-0-306-40615-7",
			wantQuery: "9780306406157",
		},
		{
			name:      "preserves ISBN-10 X check digit",
			isbn:      "0-8044-2957-x",
			wantQuery: "080442957X",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotQuery string
			client := &Client{
				httpClient: &mockHTTPClient{
					handler: func(req *http.Request) (*http.Response, error) {
						gotQuery = req.URL.Query().Get("q")
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(bytes.NewReader([]byte("[]"))),
							Header:     make(http.Header),
						}, nil
					},
				},
			}

			results, err := client.SearchByISBN(t.Context(), tt.isbn)
			require.NoError(t, err)
			require.Empty(t, results)
			require.Equal(t, tt.wantQuery, gotQuery)
		})
	}
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

	results, err := client.parseISBNSearchResponse(t.Context(), AutoComplete)
	require.NoError(t, err)
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
			results, err := client.parseISBNSearchResponse(t.Context(), []byte(tt.body))
			require.NoError(t, err)
			if len(results) != 0 {
				t.Errorf("expected 0 results, got %d", len(results))
			}
		})
	}
}

func TestParseISBNSearchResponse_InvalidBookID(t *testing.T) {
	body := `[{"bookId": "not-a-number", "workId": "456", "title": "Test"}]`
	results, err := noResponseClient().parseISBNSearchResponse(t.Context(), []byte(body))
	require.NoError(t, err)
	if len(results) != 0 {
		t.Errorf("expected 0 results for invalid bookId, got %d", len(results))
	}
}

func TestParseISBNSearchResponse_InvalidWorkID(t *testing.T) {
	body := `[{"bookId": "123", "workId": "not-a-number", "title": "Test"}]`
	results, err := noResponseClient().parseISBNSearchResponse(t.Context(), []byte(body))
	require.NoError(t, err)
	if len(results) != 0 {
		t.Errorf("expected 0 results for invalid workId, got %d", len(results))
	}
}

func TestParseISBNSearchResponse_OptionalFieldsMissing(t *testing.T) {
	body := `[{"bookId": "123", "workId": "456", "title": "Minimal Book"}]`
	results, err := noResponseClient().parseISBNSearchResponse(t.Context(), []byte(body))
	require.NoError(t, err)
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
	if r.AuthorName != "" {
		t.Errorf("AuthorName = %q, want empty", r.AuthorName)
	}
}

func TestParseISBNSearchResponse_EmptyArray(t *testing.T) {
	results, err := noResponseClient().parseISBNSearchResponse(t.Context(), []byte(`[]`))
	require.NoError(t, err)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseISBNSearchResponse_InvalidJSON(t *testing.T) {
	results, err := noResponseClient().parseISBNSearchResponse(t.Context(), []byte(`not json`))
	require.Error(t, err)
	require.Nil(t, results)
}

func TestParseISBNSearchResponse_MultipleResults(t *testing.T) {
	body := `[
		{"bookId": "111", "workId": "222", "title": "Book One", "author": {"id": 1, "name": "Author A"}},
		{"bookId": "333", "workId": "444", "title": "Book Two", "author": {"id": 2, "name": "Author B"}}
	]`
	results, err := noResponseClient().parseISBNSearchResponse(t.Context(), []byte(body))
	require.NoError(t, err)
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
	results, err := noResponseClient().parseISBNSearchResponse(t.Context(), []byte(body))
	require.NoError(t, err)
	if len(results) != 1 {
		t.Fatalf("expected 1 result (skipping invalid), got %d", len(results))
	}
	if results[0].BookTitle != "Valid Entry" {
		t.Errorf("BookTitle = %q, want %q", results[0].BookTitle, "Valid Entry")
	}
}

func TestSearchByISBN_ReturnsErrorForMalformedAutocompletePayload(t *testing.T) {
	client := &Client{
		httpClient: &mockHTTPClient{
			handler: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`not json`))),
					Header:     make(http.Header),
				}, nil
			},
		},
	}

	results, err := client.SearchByISBN(t.Context(), "9780306406157")
	require.Error(t, err)
	require.Nil(t, results)
}

func TestSearchByISBN_RejectsInvalidCharacters(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name string
		isbn string
	}{
		{name: "exclamation mark", isbn: "059313520!"},
		{name: "letter in middle", isbn: "97803064a6157"},
		{name: "X not in last position of ISBN-10", isbn: "0X93135202"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := client.SearchByISBN(t.Context(), tt.isbn)
			require.Error(t, err)
			require.Nil(t, results)
			require.Contains(t, err.Error(), "unexpected character")
		})
	}
}

func TestParseISBNSearchResponse_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately

	client := noResponseClient()
	body := `[{"bookId": "123", "workId": "456", "title": "Test Book"}]`
	results, err := client.parseISBNSearchResponse(ctx, []byte(body))
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, results)
}

func TestParseISBNSearchResponse_ConcurrentGraphQLCalls(t *testing.T) {
	// Track the number of concurrent GraphQL calls to verify fan-out.
	var callCount atomic.Int32

	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				callCount.Add(1)
				resp.Data = nil
				return errors.New("mock error for concurrency test")
			},
		},
	}

	body := `[
		{"bookId": "111", "workId": "222", "title": "Book One"},
		{"bookId": "333", "workId": "444", "title": "Book Two"},
		{"bookId": "555", "workId": "666", "title": "Book Three"}
	]`
	results, err := client.parseISBNSearchResponse(t.Context(), []byte(body))
	require.NoError(t, err)
	require.Len(t, results, 3)

	// All 3 entries should have triggered a GraphQL call.
	require.Equal(t, int32(3), callCount.Load())

	// Verify order is preserved.
	require.Equal(t, "Book One", results[0].BookTitle)
	require.Equal(t, "Book Two", results[1].BookTitle)
	require.Equal(t, "Book Three", results[2].BookTitle)
}

func TestParseISBNSearchResponse_CapsResultsAtFive(t *testing.T) {
	var callCount atomic.Int32

	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				callCount.Add(1)
				resp.Data = nil
				return errors.New("mock error for cap test")
			},
		},
	}

	// Build a JSON array with 8 entries — only the first 5 should be processed.
	body := `[
		{"bookId": "1", "workId": "10", "title": "Book One"},
		{"bookId": "2", "workId": "20", "title": "Book Two"},
		{"bookId": "3", "workId": "30", "title": "Book Three"},
		{"bookId": "4", "workId": "40", "title": "Book Four"},
		{"bookId": "5", "workId": "50", "title": "Book Five"},
		{"bookId": "6", "workId": "60", "title": "Book Six"},
		{"bookId": "7", "workId": "70", "title": "Book Seven"},
		{"bookId": "8", "workId": "80", "title": "Book Eight"}
	]`
	results, err := client.parseISBNSearchResponse(t.Context(), []byte(body))
	require.NoError(t, err)
	require.Len(t, results, 5)

	// Only 5 GraphQL calls should have been made.
	require.Equal(t, int32(5), callCount.Load())

	// Verify the first 5 entries were kept in order.
	require.Equal(t, "Book One", results[0].BookTitle)
	require.Equal(t, "Book Five", results[4].BookTitle)
}

func TestSearchByISBN_PropagatesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	client := &Client{
		httpClient: &mockHTTPClient{
			handler: func(req *http.Request) (*http.Response, error) {
				// Cancel the context after the HTTP request succeeds
				cancel()
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`[{"bookId": "123", "workId": "456", "title": "Test"}]`))),
					Header:     make(http.Header),
				}, nil
			},
		},
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				return context.Canceled
			},
		},
	}

	results, err := client.SearchByISBN(ctx, "9780306406157")
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, results)
}
