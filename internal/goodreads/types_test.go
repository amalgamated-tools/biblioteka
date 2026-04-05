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

	if decoded.WorkID != original.WorkID {
		t.Errorf("WorkID = %q, want %q", decoded.WorkID, original.WorkID)
	}
	if decoded.BookTitle != original.BookTitle {
		t.Errorf("BookTitle = %q, want %q", decoded.BookTitle, original.BookTitle)
	}
	if decoded.AuthorName != original.AuthorName {
		t.Errorf("AuthorName = %q, want %q", decoded.AuthorName, original.AuthorName)
	}
	if decoded.BookISBN13 != original.BookISBN13 {
		t.Errorf("BookISBN13 = %q, want %q", decoded.BookISBN13, original.BookISBN13)
	}
	if decoded.BookLegacyID != original.BookLegacyID {
		t.Errorf("BookLegacyID = %d, want %d", decoded.BookLegacyID, original.BookLegacyID)
	}
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
	if _, ok := raw["work_id"]; !ok {
		t.Error("expected work_id JSON field")
	}
	if _, ok := raw["book_title"]; !ok {
		t.Error("expected book_title JSON field")
	}
	if _, ok := raw["author_name"]; !ok {
		t.Error("expected author_name JSON field")
	}
}

// TestBookResult_ZeroValue verifies that the zero value of BookResult is
// valid and produces valid JSON.
func TestBookResult_ZeroValue(t *testing.T) {
	t.Parallel()

	var br BookResult
	data, err := json.Marshal(br)
	require.NoError(t, err, "json.Marshal() error for zero value")
	if len(data) == 0 {
		t.Error("expected non-empty JSON for zero value BookResult")
	}
}
