package goodreads

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBookResult_JSONRoundTrip verifies that a BookResult can be marshaled to
// JSON and unmarshaled back with all fields preserved.
func TestBookResult_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := BookResult{
		WorkID:                "W123456",
		WorkLegacyID:          789,
		BookID:                "B654321",
		BookLegacyID:          123,
		BookImageURL:          "https://example.com/cover.jpg",
		BookTitle:             "A Test Book",
		BookASIN:              "B0ABCDEF01",
		BookISBN:              "123456789X",
		BookISBN13:            "9781234567890",
		BookLanguage:          "en",
		AuthorID:              "A99999",
		AuthorName:            "Jane Doe",
		AuthorLegacyID:        42,
		AuthorProfileImageURL: "https://example.com/author.jpg",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err, "json.Marshal() error")

	var decoded BookResult
	require.NoError(t, json.Unmarshal(data, &decoded), "json.Unmarshal() error")

	require.Equal(t, original.WorkID, decoded.WorkID)
	require.Equal(t, original.BookTitle, decoded.BookTitle)
	require.Equal(t, original.AuthorName, decoded.AuthorName)
	require.Equal(t, original.BookISBN13, decoded.BookISBN13)
	require.Equal(t, original.BookLegacyID, decoded.BookLegacyID)
}

// TestBookResult_JSONFieldNames verifies that the JSON field names use
// snake_case as required by the API contract.
func TestBookResult_JSONFieldNames(t *testing.T) {
	t.Parallel()

	br := BookResult{
		WorkID:     "W1",
		BookTitle:  "My Book",
		AuthorName: "Author Name",
	}

	data, err := json.Marshal(br)
	require.NoError(t, err, "json.Marshal() error")

	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw), "json.Unmarshal() error")

	// Verify snake_case JSON keys.
	_, ok := raw["work_id"]
	require.True(t, ok, "expected work_id JSON field")
	_, ok = raw["book_title"]
	require.True(t, ok, "expected book_title JSON field")
	_, ok = raw["author_name"]
	require.True(t, ok, "expected author_name JSON field")
}

// TestBookResult_ZeroValue verifies that the zero value of BookResult is
// valid and produces valid JSON.
func TestBookResult_ZeroValue(t *testing.T) {
	t.Parallel()

	var br BookResult
	data, err := json.Marshal(br)
	require.NoError(t, err, "json.Marshal() error for zero value")
	require.NotEmpty(t, data)
}
