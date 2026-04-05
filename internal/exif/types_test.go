package exif

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestExifToolOutput_ISBN_PrefersISBN13 verifies that ISBN() returns the
// ISBN13 value when both ISBN13 and ISBN10 are set.
func TestExifToolOutput_ISBN_PrefersISBN13(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		ISBN13: "9781234567890",
		ISBN10: "123456789X",
	}
	got := out.ISBN()
	require.Equal(t, "9781234567890", got, "ISBN13 should take precedence")
}

// TestExifToolOutput_ISBN_FallsBackToISBN10 verifies that ISBN() returns
// ISBN10 when ISBN13 is empty.
func TestExifToolOutput_ISBN_FallsBackToISBN10(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{ISBN10: "123456789X"}
	got := out.ISBN()
	require.Equal(t, "123456789X", got)
}

// TestExifToolOutput_ISBN_EmptyWhenNeither verifies that ISBN() returns an
// empty string when both fields are empty.
func TestExifToolOutput_ISBN_EmptyWhenNeither(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{}
	got := out.ISBN()
	require.Equal(t, "", got)
}

// TestExifToolOutput_SetISBN_13Digit verifies that SetISBN with a 13-digit
// value sets ISBN13 and clears ISBN10.
func TestExifToolOutput_SetISBN_13Digit(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{ISBN10: "123456789X"}
	out.SetISBN("9781234567890")
	require.Equal(t, "9781234567890", out.ISBN13)
	require.Equal(t, "", out.ISBN10, "should be cleared")
}

// TestExifToolOutput_SetISBN_10Digit verifies that SetISBN with a 10-digit
// value sets ISBN10 and clears ISBN13.
func TestExifToolOutput_SetISBN_10Digit(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{ISBN13: "9781234567890"}
	out.SetISBN("123456789X")
	require.Equal(t, "123456789X", out.ISBN10)
	require.Equal(t, "", out.ISBN13, "should be cleared")
}

// TestExifToolOutput_SetISBN_EmptyClears verifies that SetISBN("") clears
// both ISBN fields.
func TestExifToolOutput_SetISBN_EmptyClears(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		ISBN13: "9781234567890",
		ISBN10: "123456789X",
	}
	out.SetISBN("")
	require.Equal(t, "", out.ISBN13)
	require.Equal(t, "", out.ISBN10)
}

// TestExifToolOutput_SetISBN_UnknownLengthIsNoop verifies that SetISBN with
// a value that is neither 0, 10, nor 13 characters long is a no-op.
func TestExifToolOutput_SetISBN_UnknownLengthIsNoop(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		ISBN13: "9781234567890",
		ISBN10: "123456789X",
	}
	out.SetISBN("12345") // 5-character value — should not change anything
	require.Equal(t, "9781234567890", out.ISBN13)
	require.Equal(t, "123456789X", out.ISBN10)
}

// TestExifToolOutput_SetISBN_RoundTrip verifies that SetISBN followed by
// ISBN() returns the original value.
func TestExifToolOutput_SetISBN_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "ISBN-13", input: "9781234567890"},
		{name: "ISBN-10", input: "123456789X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := &ExifToolOutput{}
			out.SetISBN(tt.input)
			got := out.ISBN()
			require.Equal(t, tt.input, got, "round-trip: SetISBN(%q) then ISBN()", tt.input)
		})
	}
}
