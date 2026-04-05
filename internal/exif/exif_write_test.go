package exif

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHandleWriteMetadataResponse_Success verifies that the response with
// the success token produces no error.
func TestHandleWriteMetadataResponse_Success(t *testing.T) {
	t.Parallel()

	err := handleWriteMetadataResponse("  1 image files updated\n")
	require.NoError(t, err)
}

// TestHandleWriteMetadataResponse_ErrorMessage verifies that a non-success
// response produces an error with the trimmed response as message.
func TestHandleWriteMetadataResponse_ErrorMessage(t *testing.T) {
	t.Parallel()

	err := handleWriteMetadataResponse("  Error writing file\n")
	require.Error(t, err, "expected error for non-success response")
	require.NotEmpty(t, err.Error())
}

// TestHandleWriteMetadataResponse_PartialMatch verifies that a partial match
// of the success token is not treated as a success.
func TestHandleWriteMetadataResponse_PartialMatch(t *testing.T) {
	t.Parallel()

	// Not ending with the success token.
	err := handleWriteMetadataResponse("image files updated")
	require.Error(t, err, "expected error for partial success token")
}

// TestToString_StringPassthrough verifies that a string value passes through
// unchanged.
func TestToString_StringPassthrough(t *testing.T) {
	t.Parallel()

	got := toString("hello world")
	require.Equal(t, "hello world", got)
}

// TestToString_Float64 verifies that a float64 is converted correctly.
func TestToString_Float64(t *testing.T) {
	t.Parallel()

	got := toString(float64(3.14))
	require.Equal(t, "3.14", got)
}

// TestToString_Int64 verifies that an int64 is converted correctly.
func TestToString_Int64(t *testing.T) {
	t.Parallel()

	got := toString(int64(42))
	require.Equal(t, "42", got)
}

// TestToString_Float64WholeNumber verifies that a whole-number float64 is
// rendered without a decimal point.
func TestToString_Float64WholeNumber(t *testing.T) {
	t.Parallel()

	got := toString(float64(7))
	require.Equal(t, "7", got)
}

// TestEmptyFileMetadata verifies that EmptyFileMetadata creates a struct with
// an initialized (non-nil) Fields map.
func TestEmptyFileMetadata(t *testing.T) {
	t.Parallel()

	fm := EmptyFileMetadata()
	require.NotNil(t, fm.Fields, "expected non-nil Fields map in EmptyFileMetadata()")
}

// TestFileMetadataSetStringAndGetStrings verifies the round-trip: set a string
// value then retrieve it via GetStrings.
func TestFileMetadataSetStringAndGetStrings(t *testing.T) {
	t.Parallel()

	fm := EmptyFileMetadata()
	fm.SetString("Title", "My Test Title")

	vals, err := fm.GetStrings("Title")
	require.NoError(t, err, "GetStrings() error")
	require.Len(t, vals, 1)
	require.Equal(t, "My Test Title", vals[0])
}

// TestFileMetadataGetStrings_KeyNotFound verifies that GetStrings returns
// ErrKeyNotFound for a missing key.
func TestFileMetadataGetStrings_KeyNotFound(t *testing.T) {
	t.Parallel()

	fm := EmptyFileMetadata()
	_, err := fm.GetStrings("NonExistentKey")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

// TestFileMetadataSetString_MultipleFields verifies that multiple fields can
// be set and retrieved independently.
func TestFileMetadataSetString_MultipleFields(t *testing.T) {
	t.Parallel()

	fm := EmptyFileMetadata()
	fm.SetString("Author", "Jane Doe")
	fm.SetString("Title", "Great Book")

	authors, err := fm.GetStrings("Author")
	require.NoError(t, err)
	require.Len(t, authors, 1)
	require.Equal(t, "Jane Doe", authors[0])

	titles, err := fm.GetStrings("Title")
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.Equal(t, "Great Book", titles[0])
}
