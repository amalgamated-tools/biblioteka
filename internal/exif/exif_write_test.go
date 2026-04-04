package exif

import (
	"testing"
)

// TestHandleWriteMetadataResponse_Success verifies that the response with
// the success token produces no error.
func TestHandleWriteMetadataResponse_Success(t *testing.T) {
	t.Parallel()

	err := handleWriteMetadataResponse("  1 image files updated\n")
	if err != nil {
		t.Errorf("handleWriteMetadataResponse() unexpected error: %v", err)
	}
}

// TestHandleWriteMetadataResponse_ErrorMessage verifies that a non-success
// response produces an error with the trimmed response as message.
func TestHandleWriteMetadataResponse_ErrorMessage(t *testing.T) {
	t.Parallel()

	err := handleWriteMetadataResponse("  Error writing file\n")
	if err == nil {
		t.Fatal("expected error for non-success response")
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

// TestHandleWriteMetadataResponse_PartialMatch verifies that a partial match
// of the success token is not treated as a success.
func TestHandleWriteMetadataResponse_PartialMatch(t *testing.T) {
	t.Parallel()

	// Not ending with the success token.
	err := handleWriteMetadataResponse("image files updated")
	if err == nil {
		t.Error("expected error for partial success token")
	}
}

// TestToString_StringPassthrough verifies that a string value passes through
// unchanged.
func TestToString_StringPassthrough(t *testing.T) {
	t.Parallel()

	got := toString("hello world")
	if got != "hello world" {
		t.Errorf("toString(%q) = %q, want %q", "hello world", got, "hello world")
	}
}

// TestToString_Float64 verifies that a float64 is converted correctly.
func TestToString_Float64(t *testing.T) {
	t.Parallel()

	got := toString(float64(3.14))
	if got != "3.14" {
		t.Errorf("toString(3.14) = %q, want 3.14", got)
	}
}

// TestToString_Int64 verifies that an int64 is converted correctly.
func TestToString_Int64(t *testing.T) {
	t.Parallel()

	got := toString(int64(42))
	if got != "42" {
		t.Errorf("toString(42) = %q, want 42", got)
	}
}

// TestToString_Float64WholeNumber verifies that a whole-number float64 is
// rendered without a decimal point.
func TestToString_Float64WholeNumber(t *testing.T) {
	t.Parallel()

	got := toString(float64(7))
	if got != "7" {
		t.Errorf("toString(float64(7)) = %q, want 7", got)
	}
}

// TestEmptyFileMetadata verifies that EmptyFileMetadata creates a struct with
// an initialized (non-nil) Fields map.
func TestEmptyFileMetadata(t *testing.T) {
	t.Parallel()

	fm := EmptyFileMetadata()
	if fm.Fields == nil {
		t.Error("expected non-nil Fields map in EmptyFileMetadata()")
	}
}

// TestFileMetadataSetStringAndGetStrings verifies the round-trip: set a string
// value then retrieve it via GetStrings.
func TestFileMetadataSetStringAndGetStrings(t *testing.T) {
	t.Parallel()

	fm := EmptyFileMetadata()
	fm.SetString("Title", "My Test Title")

	vals, err := fm.GetStrings("Title")
	if err != nil {
		t.Fatalf("GetStrings() error: %v", err)
	}
	if len(vals) != 1 || vals[0] != "My Test Title" {
		t.Errorf("GetStrings(Title) = %v, want [My Test Title]", vals)
	}
}

// TestFileMetadataGetStrings_KeyNotFound verifies that GetStrings returns
// ErrKeyNotFound for a missing key.
func TestFileMetadataGetStrings_KeyNotFound(t *testing.T) {
	t.Parallel()

	fm := EmptyFileMetadata()
	_, err := fm.GetStrings("NonExistentKey")
	if err == nil {
		t.Fatal("expected ErrKeyNotFound for missing key")
	}
	if err != ErrKeyNotFound {
		t.Errorf("err = %v, want ErrKeyNotFound", err)
	}
}

// TestFileMetadataSetString_MultipleFields verifies that multiple fields can
// be set and retrieved independently.
func TestFileMetadataSetString_MultipleFields(t *testing.T) {
	t.Parallel()

	fm := EmptyFileMetadata()
	fm.SetString("Author", "Jane Doe")
	fm.SetString("Title", "Great Book")

	authors, err := fm.GetStrings("Author")
	if err != nil || len(authors) != 1 || authors[0] != "Jane Doe" {
		t.Errorf("GetStrings(Author) = %v, %v, want [Jane Doe], nil", authors, err)
	}

	titles, err := fm.GetStrings("Title")
	if err != nil || len(titles) != 1 || titles[0] != "Great Book" {
		t.Errorf("GetStrings(Title) = %v, %v, want [Great Book], nil", titles, err)
	}
}
