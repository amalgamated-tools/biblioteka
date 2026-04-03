package exif

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeISBN(t *testing.T) {
	t.Parallel()

	require.Equal(t, "123456789X", NormalizeISBN(" urn:isbn:123-456-789x "))
	require.Equal(t, "9781234567890", NormalizeISBN("isbn: 978-1-234-56789-0"))
	require.Empty(t, NormalizeISBN("not-an-isbn"))
}

func TestExifToolOutputSetISBN(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{}

	out.SetISBN("1234567890")
	require.Equal(t, "1234567890", out.ISBN())
	require.Equal(t, "1234567890", out.ISBN10)
	require.Empty(t, out.ISBN13)

	out.SetISBN("9781234567890")
	require.Equal(t, "9781234567890", out.ISBN())
	require.Empty(t, out.ISBN10)
	require.Equal(t, "9781234567890", out.ISBN13)

	out.SetISBN("")
	require.Empty(t, out.ISBN())
	require.Empty(t, out.ISBN10)
	require.Empty(t, out.ISBN13)
}
