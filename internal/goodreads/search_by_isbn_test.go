package goodreads

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSearchByISBN_EmptyISBN verifies that an empty ISBN returns an error immediately
// without making any network requests.
func TestSearchByISBN_EmptyISBN(t *testing.T) {
	client := &Client{}
	results, err := client.SearchByISBN(t.Context(), "")
	require.Error(t, err)
	require.Nil(t, results)
	require.Contains(t, err.Error(), "ISBN cannot be empty")
}

// TestSearchByISBN_InvalidLength verifies that ISBNs with lengths other than 10 or 13
// digits are rejected.
func TestSearchByISBN_InvalidLength(t *testing.T) {
	client := &Client{}
	tests := []struct {
		name string
		isbn string
	}{
		{name: "9 digits", isbn: "978030640"},
		{name: "11 digits", isbn: "97803064061"},
		{name: "12 digits", isbn: "978030640615"},
		{name: "14 digits", isbn: "97803064061577"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := client.SearchByISBN(t.Context(), tt.isbn)
			require.Error(t, err)
			require.Nil(t, results)
		})
	}
}

// TestSearchByISBN_InvalidISBN10CheckDigit verifies that a 10-digit ISBN with a bad
// check digit is rejected before any HTTP call is made.
func TestSearchByISBN_InvalidISBN10CheckDigit(t *testing.T) {
	client := &Client{}
	// 0306406152 is valid; 0306406153 has a wrong check digit.
	results, err := client.SearchByISBN(t.Context(), "0306406153")
	require.Error(t, err)
	require.Nil(t, results)
	require.Contains(t, err.Error(), "invalid ISBN-10 check digit")
}

// TestSearchByISBN_InvalidISBN13CheckDigit verifies that a 13-digit ISBN with a bad
// check digit is rejected before any HTTP call is made.
func TestSearchByISBN_InvalidISBN13CheckDigit(t *testing.T) {
	client := &Client{}
	// 9780306406157 is valid; 9780306406158 has a wrong check digit.
	results, err := client.SearchByISBN(t.Context(), "9780306406158")
	require.Error(t, err)
	require.Nil(t, results)
	require.Contains(t, err.Error(), "invalid ISBN-13 check digit")
}

// TestSearchByISBN_HTTPFailure verifies that a network-level error is surfaced correctly.
func TestSearchByISBN_HTTPFailure(t *testing.T) {
	client := &Client{
		httpClient: &mockHTTPClient{
			handler: func(req *http.Request) (*http.Response, error) {
				return nil, &networkError{msg: "connection refused"}
			},
		},
	}
	results, err := client.SearchByISBN(t.Context(), "9780306406157")
	require.Error(t, err)
	require.Nil(t, results)
	require.Contains(t, err.Error(), "HTTP request failed")
}

// TestSearchByISBN_NonOKStatus verifies that non-200 HTTP responses are treated as errors.
func TestSearchByISBN_NonOKStatus(t *testing.T) {
	statuses := []struct {
		name   string
		status int
	}{
		{name: "not found", status: http.StatusNotFound},
		{name: "internal server error", status: http.StatusInternalServerError},
		{name: "too many requests", status: http.StatusTooManyRequests},
	}
	for _, tt := range statuses {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				httpClient: &mockHTTPClient{
					handler: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: tt.status,
							Body:       io.NopCloser(bytes.NewReader(nil)),
							Header:     make(http.Header),
						}, nil
					},
				},
			}
			results, err := client.SearchByISBN(t.Context(), "9780306406157")
			require.Error(t, err)
			require.Nil(t, results)
		})
	}
}

// TestSearchByISBN_ResponseTooLarge verifies that a response body exceeding 1 MB is
// rejected without panicking or returning partial data.
func TestSearchByISBN_ResponseTooLarge(t *testing.T) {
	// Intentionally duplicates the unexported maxResponseSize in search_by_isbn.go.
	// If the production limit changes, this test must be updated to match.
	const maxResponseSize = 1 << 20 // 1 MB
	largeBody := bytes.Repeat([]byte("x"), maxResponseSize+2)
	client := &Client{
		httpClient: &mockHTTPClient{
			handler: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(largeBody)),
					Header:     make(http.Header),
				}, nil
			},
		},
	}
	results, err := client.SearchByISBN(t.Context(), "9780306406157")
	require.Error(t, err)
	require.Nil(t, results)
	require.Contains(t, err.Error(), "too large")
}

// TestParseAutocompleteEntries_ValidEntry verifies that a well-formed autocomplete
// JSON array is parsed into the expected slice of autocompleteEntry values.
func TestParseAutocompleteEntries_ValidEntry(t *testing.T) {
	body := []byte(`[{"bookId":"54493401","workId":"79106958","title":"Project Hail Mary","imageUrl":"https://example.com/img.jpg","author":{"id":6540057,"name":"Andy Weir"}}]`)
	entries, err := parseAutocompleteEntries(t.Context(), body)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	e := entries[0]
	require.Equal(t, int64(54493401), e.bookID)
	require.Equal(t, int64(79106958), e.workID)
	require.Equal(t, "Project Hail Mary", e.title)
	require.Equal(t, "https://example.com/img.jpg", e.imageURL)
	require.Equal(t, int64(6540057), e.authorID)
	require.Equal(t, "Andy Weir", e.authorName)
}

// TestParseAutocompleteEntries_EmptyArray verifies that an empty JSON array produces
// an empty slice without error.
func TestParseAutocompleteEntries_EmptyArray(t *testing.T) {
	entries, err := parseAutocompleteEntries(t.Context(), []byte(`[]`))
	require.NoError(t, err)
	require.Empty(t, entries)
}

// TestParseAutocompleteEntries_InvalidJSON verifies that malformed JSON is surfaced as an
// error.
func TestParseAutocompleteEntries_InvalidJSON(t *testing.T) {
	entries, err := parseAutocompleteEntries(t.Context(), []byte(`not json`))
	require.Error(t, err)
	require.Nil(t, entries)
}

// TestParseAutocompleteEntries_MissingOptionalFields verifies that optional fields
// (imageUrl, author) default to zero values without error.
func TestParseAutocompleteEntries_MissingOptionalFields(t *testing.T) {
	body := []byte(`[{"bookId":"123","workId":"456","title":"Minimal"}]`)
	entries, err := parseAutocompleteEntries(t.Context(), body)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	e := entries[0]
	require.Equal(t, int64(123), e.bookID)
	require.Equal(t, int64(456), e.workID)
	require.Equal(t, "Minimal", e.title)
	require.Empty(t, e.imageURL)
	require.Zero(t, e.authorID)
	require.Empty(t, e.authorName)
}

// TestParseAutocompleteEntries_SkipsEntriesWithMissingRequiredFields verifies that
// entries missing bookId, workId, or title are silently dropped.
func TestParseAutocompleteEntries_SkipsEntriesWithMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing bookId", body: `[{"workId":"456","title":"Test"}]`},
		{name: "missing workId", body: `[{"bookId":"123","title":"Test"}]`},
		{name: "missing title", body: `[{"bookId":"123","workId":"456"}]`},
		{name: "zero bookId", body: `[{"bookId":"0","workId":"456","title":"Test"}]`},
		{name: "zero workId", body: `[{"bookId":"123","workId":"0","title":"Test"}]`},
		{name: "empty title", body: `[{"bookId":"123","workId":"456","title":""}]`},
		{name: "non-numeric bookId", body: `[{"bookId":"abc","workId":"456","title":"Test"}]`},
		{name: "non-numeric workId", body: `[{"bookId":"123","workId":"abc","title":"Test"}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := parseAutocompleteEntries(t.Context(), []byte(tt.body))
			require.NoError(t, err)
			require.Empty(t, entries)
		})
	}
}

// TestParseAutocompleteEntries_MultipleEntries verifies that multiple valid entries are
// all returned in order.
func TestParseAutocompleteEntries_MultipleEntries(t *testing.T) {
	body := []byte(`[
		{"bookId":"1","workId":"10","title":"Book One","author":{"id":100,"name":"Author A"}},
		{"bookId":"2","workId":"20","title":"Book Two","author":{"id":200,"name":"Author B"}}
	]`)
	entries, err := parseAutocompleteEntries(t.Context(), body)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "Book One", entries[0].title)
	require.Equal(t, int64(1), entries[0].bookID)
	require.Equal(t, "Author A", entries[0].authorName)
	require.Equal(t, "Book Two", entries[1].title)
	require.Equal(t, int64(200), entries[1].authorID)
}

// TestParseAutocompleteEntries_UsesRealFixture verifies that the real autocomplete.json
// fixture is parsed without error and produces the expected number of entries.
func TestParseAutocompleteEntries_UsesRealFixture(t *testing.T) {
	entries, err := parseAutocompleteEntries(t.Context(), AutoComplete)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "Project Hail Mary", entries[0].title)
	require.Equal(t, int64(54493401), entries[0].bookID)
	require.Equal(t, int64(79106958), entries[0].workID)
}

// TestBuildFallbackResult verifies that buildFallbackResult converts an autocompleteEntry
// into the expected BookResult, including string-formatted legacy IDs.
func TestBuildFallbackResult(t *testing.T) {
	e := autocompleteEntry{
		bookID:     54493401,
		workID:     79106958,
		title:      "Project Hail Mary",
		imageURL:   "https://example.com/img.jpg",
		authorID:   6540057,
		authorName: "Andy Weir",
	}
	r := buildFallbackResult(e)
	require.Equal(t, "54493401", r.BookID)
	require.Equal(t, "79106958", r.WorkID)
	require.Equal(t, int64(54493401), r.BookLegacyID)
	require.Equal(t, int64(79106958), r.WorkLegacyID)
	require.Equal(t, "Project Hail Mary", r.BookTitle)
	require.Equal(t, "https://example.com/img.jpg", r.BookImageURL)
	require.Equal(t, int64(6540057), r.AuthorLegacyID)
	require.Equal(t, "Andy Weir", r.AuthorName)
	// Fields not in autocomplete data should be empty.
	require.Empty(t, r.BookASIN)
	require.Empty(t, r.BookISBN)
	require.Empty(t, r.BookISBN13)
	require.Empty(t, r.BookLanguage)
	require.Empty(t, r.AuthorID)
	require.Empty(t, r.AuthorProfileImageURL)
}

// TestBuildFallbackResult_ZeroEntry verifies that a zero-valued autocompleteEntry
// produces a BookResult with string "0" IDs and all other fields empty.
func TestBuildFallbackResult_ZeroEntry(t *testing.T) {
	r := buildFallbackResult(autocompleteEntry{})
	require.Equal(t, "0", r.BookID)
	require.Equal(t, "0", r.WorkID)
	require.Zero(t, r.BookLegacyID)
	require.Zero(t, r.WorkLegacyID)
	require.Empty(t, r.BookTitle)
	require.Empty(t, r.BookImageURL)
	require.Empty(t, r.AuthorName)
	require.Zero(t, r.AuthorLegacyID)
}

// networkError is a minimal error type used to simulate network failures in tests.
type networkError struct{ msg string }

func (e *networkError) Error() string { return e.msg }

// TestSearchByISBN_BuildsCorrectURL verifies that SearchByISBN constructs the expected
// auto_complete URL with the normalized ISBN as the query parameter.
func TestSearchByISBN_BuildsCorrectURL(t *testing.T) {
	var capturedURL string
	client := &Client{
		httpClient: &mockHTTPClient{
			handler: func(req *http.Request) (*http.Response, error) {
				capturedURL = req.URL.String()
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`[]`))),
					Header:     make(http.Header),
				}, nil
			},
		},
	}
	_, err := client.SearchByISBN(t.Context(), "9780306406157")
	require.NoError(t, err)
	require.Contains(t, capturedURL, "goodreads.com/book/auto_complete")
	require.Contains(t, capturedURL, "9780306406157")
}
