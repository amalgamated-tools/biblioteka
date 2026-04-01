package exif

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTSV_EmptyInput(t *testing.T) {
	out, err := ParseTSV(t.Context(), "", "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if len(out.Identifiers) != 0 {
		t.Errorf("expected 0 identifiers, got %d", len(out.Identifiers))
	}
}

func TestParseTSV_IdentifierWithoutScheme(t *testing.T) {
	input := "Identifier\turn:isbn:1234567890\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if got := len(out.Identifiers); got != 1 {
		t.Fatalf("expected 1 identifier, got %d", got)
	}
	require.Equal(t, out.Identifiers[0].Value, "urn:isbn:1234567890", "Identifiers[0].Value")
	require.Equal(t, out.Identifiers[0].Scheme, "", "Identifiers[0].Scheme")
}

func TestParseTSV_IdentifierWithScheme(t *testing.T) {
	input := "Identifier Scheme\tAMAZON\nIdentifier\tB08FHBV4ZX\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if got := len(out.Identifiers); got != 1 {
		t.Fatalf("expected 1 identifier, got %d", got)
	}
	require.Equal(t, out.Identifiers[0].Value, "B08FHBV4ZX", "Identifiers[0].Value")
	require.Equal(t, out.Identifiers[0].Scheme, "AMAZON", "Identifiers[0].Scheme")
}

func TestParseTSV_IdentifierIdPrecedesValue(t *testing.T) {
	input := "Identifier Id\tuid\nIdentifier\t12345\nIdentifier Scheme\tcalibre\nIdentifier\tabcdef\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if got := len(out.Identifiers); got != 2 {
		t.Fatalf("expected 2 identifiers, got %d", got)
	}
	require.Equal(t, out.Identifiers[0].Value, "12345", "Identifiers[0].Value")
	require.Equal(t, out.Identifiers[0].ID, "uid", "Identifiers[0].ID")
	require.Equal(t, out.Identifiers[0].Scheme, "", "Identifiers[0].Scheme")
	require.Equal(t, out.Identifiers[1].Value, "abcdef", "Identifiers[1].Value")
	require.Equal(t, out.Identifiers[1].Scheme, "calibre", "Identifiers[1].Scheme")
	require.Equal(t, out.Identifiers[1].ID, "", "Identifiers[1].ID")
}

func TestParseTSV_MultipleIdentifiers(t *testing.T) {
	input := "Identifier Scheme\tISBN\nIdentifier\tAAA\nIdentifier Scheme\tAMAZON\nIdentifier\tBBB\nIdentifier\tCCC\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if got := len(out.Identifiers); got != 3 {
		t.Fatalf("expected 3 identifiers, got %d", got)
	}
	require.Equal(t, out.Identifiers[0].Value, "AAA", "Identifiers[0].Value")
	require.Equal(t, out.Identifiers[0].Scheme, "ISBN", "Identifiers[0].Scheme")
	require.Equal(t, out.Identifiers[1].Value, "BBB", "Identifiers[1].Value")
	require.Equal(t, out.Identifiers[1].Scheme, "AMAZON", "Identifiers[1].Scheme")
	require.Equal(t, out.Identifiers[2].Value, "CCC", "Identifiers[2].Value")
	require.Equal(t, out.Identifiers[2].Scheme, "", "Identifiers[2].Scheme")
}

func TestParseTSV_MetaPairs(t *testing.T) {
	input := "Meta Content\tcover\nMeta Name\tcover\nMeta Content\tA Novel\nMeta Name\tbooklore:subtitle\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if got := len(out.MetaTags); got != 2 {
		t.Fatalf("expected 2 meta tags, got %d", got)
	}
	require.Equal(t, out.MetaTags[0].Content, "cover", "MetaTags[0].Content")
	require.Equal(t, out.MetaTags[0].Name, "cover", "MetaTags[0].Name")
	require.Equal(t, out.MetaTags[1].Content, "A Novel", "MetaTags[1].Content")
	require.Equal(t, out.MetaTags[1].Name, "booklore:subtitle", "MetaTags[1].Name")
}

func TestParseTSV_UnknownFieldsGoToExtra(t *testing.T) {
	input := "ExifTool Version Number\t13.50\nFile Permissions\t-rw-r--r--\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	require.Equal(t, out.Extra["ExifTool Version Number"], "13.50", "Extra[ExifTool Version Number]")
	require.Equal(t, out.Extra["File Permissions"], "-rw-r--r--", "Extra[File Permissions]")
}
