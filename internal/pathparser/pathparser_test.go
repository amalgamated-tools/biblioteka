package pathparser

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBookPath(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name        string
		filePath    string
		libraryRoot string
		want        PathInfo
	}{
		{
			name: "3-level: author/series/numbered file with author and year",
			filePath: filepath.Join(
				root,
				"Alexander McCall Smith",
				"No. 1 Ladies' Detective Agency",
				"10. Tea Time for the Traditionally Built - Alexander McCall Smith (2009).epub",
			),
			want: PathInfo{
				Author:         "Alexander McCall Smith",
				Title:          "Tea Time for the Traditionally Built",
				SeriesName:     "No. 1 Ladies' Detective Agency",
				SeriesPosition: new(float64(10)),
				Year:           new(int(2009)),
			},
		},
		{
			name: "3-level: second book in series",
			filePath: filepath.Join(
				root,
				"Alexander McCall Smith",
				"No. 1 Ladies' Detective Agency",
				"11. The Double Comfort Safari Club - Alexander McCall Smith (2010).epub",
			),
			want: PathInfo{
				Author:         "Alexander McCall Smith",
				Title:          "The Double Comfort Safari Club",
				SeriesName:     "No. 1 Ladies' Detective Agency",
				SeriesPosition: new(float64(11)),
				Year:           new(int(2010)),
			},
		},
		{
			name: "2-level: author folder with numbered file",
			filePath: filepath.Join(
				root,
				"Agatha Christie",
				"1. The Seven Dials Mystery - Agatha Christie (2010).epub",
			),
			want: PathInfo{
				Author:         "Agatha Christie",
				Title:          "The Seven Dials Mystery",
				SeriesPosition: new(float64(1)),
				Year:           new(int(2010)),
			},
		},
		{
			name:     "flat file: author dash title",
			filePath: filepath.Join(root, "Mary Shelley - Frankenstein.epub"),
			want: PathInfo{
				Author: "Mary Shelley",
				Title:  "Frankenstein",
			},
		},
		{
			name:     "flat file: author dash title (ambiguous)",
			filePath: filepath.Join(root, "Moby Dick - Herman Melville.epub"),
			want: PathInfo{
				Author: "Moby Dick",
				Title:  "Herman Melville",
			},
		},
		{
			name: "1-level: author-title folder with matching filename",
			filePath: filepath.Join(
				root,
				"Emily Brontë - Wuthering Heights",
				"Emily Brontë - Wuthering Heights.epub",
			),
			want: PathInfo{
				Author: "Emily Brontë",
				Title:  "Wuthering Heights",
			},
		},
		{
			name:        "flat file: no dash separator",
			filePath:    "/library/Frankenstein.epub",
			libraryRoot: "/library",
			want: PathInfo{
				Title: "Frankenstein",
			},
		},
		{
			name:        "2-level: simple author/title",
			filePath:    "/library/Jane Austen/Pride and Prejudice.epub",
			libraryRoot: "/library",
			want: PathInfo{
				Author: "Jane Austen",
				Title:  "Pride and Prejudice",
			},
		},
		{
			name:        "2-level: filename with year only",
			filePath:    "/library/Jane Austen/Pride and Prejudice (1813).epub",
			libraryRoot: "/library",
			want: PathInfo{
				Author: "Jane Austen",
				Title:  "Pride and Prejudice",
				Year:   new(int(1813)),
			},
		},
		{
			name:        "Calibre-style: Author/Title/file.epub does not create phantom series",
			filePath:    "/library/Jane Austen/Pride and Prejudice/book.epub",
			libraryRoot: "/library",
			want: PathInfo{
				Author: "Jane Austen",
				Title:  "book",
			},
		},
		{
			name:        "3-level: series dir only used when filename has position prefix",
			filePath:    "/library/Brandon Sanderson/Mistborn/2. The Well of Ascension.epub",
			libraryRoot: "/library",
			want: PathInfo{
				Author:         "Brandon Sanderson",
				Title:          "The Well of Ascension",
				SeriesName:     "Mistborn",
				SeriesPosition: new(float64(2)),
			},
		},
		{
			name:        "flat 'X - Y' filename: left part is Author, right part is Title",
			filePath:    "/library/Tea Time for the Traditionally Built - Jean-Paul Sartre.epub",
			libraryRoot: "/library",
			want: PathInfo{
				Author: "Tea Time for the Traditionally Built",
				Title:  "Jean-Paul Sartre",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lr := root
			if tt.libraryRoot != "" {
				lr = tt.libraryRoot
			}
			got := ParseBookPath(tt.filePath, lr)
			require.Equal(t, tt.want.Author, got.Author, "Author")
			require.Equal(t, tt.want.Title, got.Title, "Title")
			require.Equal(t, tt.want.SeriesName, got.SeriesName, "SeriesName")
			require.True(t, float64PtrEqual(got.SeriesPosition, tt.want.SeriesPosition),
				"SeriesPosition: got %s, want %s", fmtF64(got.SeriesPosition), fmtF64(tt.want.SeriesPosition))
			require.True(t, intPtrEqual(got.Year, tt.want.Year),
				"Year: got %s, want %s", fmtInt(got.Year), fmtInt(tt.want.Year))
		})
	}
}

// --- isLikelyPersonName ---

func TestIsLikelyPersonName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid names
		{"two words", "Jane Doe", true},
		{"two words alt", "Isaac Asimov", true},
		{"two words alt2", "Mary Shelley", true},
		{"three words", "Alexander McCall Smith", true},
		{"four words (max)", "Jean Paul Sartre Jr", true},

		// Too few words
		{"empty string", "", false},
		{"single word", "Shakespeare", false},
		{"blank whitespace", "   ", false},

		// Too many words (> 4)
		{"five words", "One Two Three Four Five", false},

		// Contains digits
		{"digit at end", "Jane Doe 2", false},
		{"digit in word", "H3nry Ford", false},

		// Leading articles (common subtitle patterns)
		{"leading a", "A Novel", false},
		{"leading an", "An Introduction", false},
		{"leading the", "The Great Adventure", false},

		// Word not starting with uppercase letter
		{"lowercase first", "jane Doe", false},
		{"lowercase second", "Jane doe", false},
		{"hyphen lowercase", "jean-Paul Sartre", false},

		// Unicode names starting with uppercase
		{"unicode accented first", "Émile Zola", true},
		{"unicode accented first alt", "Óscar Wilde", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLikelyPersonName(tt.input)
			require.Equal(t, tt.want, got, "isLikelyPersonName(%q)", tt.input)
		})
	}
}

// --- stripTrailingAuthor ---

func TestStripTrailingAuthor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Strips when suffix looks like a person name
		{"two-word author name", "Tea Time for the Traditionally Built - Alexander McCall Smith", "Tea Time for the Traditionally Built"},
		{"two-word author name alt", "Frankenstein - Mary Shelley", "Frankenstein"},

		// Does not strip when suffix is not a person name
		{"leading article blocks strip", "Title - A Novel", "Title - A Novel"},
		{"single word not a name", "Title - Unabridged", "Title - Unabridged"},

		// Two-word capitalized suffixes are treated as person names (heuristic
		// false positive — known limitation of the simple rule-based approach).
		{"two-word capitalized suffix (false positive)", "Title - Special Edition", "Title"},

		// Does not strip when there is no " - " separator
		{"no separator", "No Separator Here", "No Separator Here"},
		{"parenthesized year", "Title (2009)", "Title (2009)"},

		// Empty suffix after " - " is left unchanged
		{"empty suffix", "Title - ", "Title - "},

		// Single-word suffix (not a person name) is left unchanged
		{"single word surname only", "Harry Potter - Rowling", "Harry Potter - Rowling"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripTrailingAuthor(tt.input)
			require.Equal(t, tt.want, got, "stripTrailingAuthor(%q)", tt.input)
		})
	}
}

// --- extractSeriesPosition ---

func TestExtractSeriesPosition(t *testing.T) {
	tests := []struct {
		input string
		want  *float64
	}{
		{"10. Tea Time for the Traditionally Built", new(float64(10))},
		{"1. The Seven Dials Mystery", new(float64(1))},
		{"2. Something", new(float64(2))},
		// No leading position prefix
		{"Tea Time for the Traditionally Built", nil},
		{"The Book", nil},
		// Non-numeric prefix
		{"abc. Title", nil},
		// Decimal/fractional format (e.g. "1.5. Title") is not supported by the
		// regex — after "1." comes "5", not whitespace, so the match fails.
		{"1.5. The Name of the Wind", nil},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractSeriesPosition(tt.input)
			require.True(t, float64PtrEqual(got, tt.want),
				"extractSeriesPosition(%q): got %s, want %s", tt.input, fmtF64(got), fmtF64(tt.want))
		})
	}
}

// --- extractYear ---

func TestExtractYear(t *testing.T) {
	tests := []struct {
		input string
		want  *int
	}{
		{"Tea Time for the Traditionally Built - Alexander McCall Smith (2009)", new(int(2009))},
		{"Pride and Prejudice (1813)", new(int(1813))},
		// No year
		{"The Book", nil},
		// Trailing spaces after year
		{"Title (2000)  ", new(int(2000))},
		// Non-4-digit number in parens does not match
		{"Title (99)", nil},
		{"Title (12345)", nil},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractYear(tt.input)
			require.True(t, intPtrEqual(got, tt.want),
				"extractYear(%q): got %s, want %s", tt.input, fmtInt(got), fmtInt(tt.want))
		})
	}
}

// --- normalizeName ---

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"two words", "Jane Doe", "janedoe"},
		{"lowercase", "jane doe", "janedoe"},
		{"extra spaces", "  Jane  Doe  ", "janedoe"},
		{"apostrophe", "Jane's Cousin", "janescousin"},
		{"hyphen", "Mary-Shelley", "maryshelley"},
		{"empty string", "", ""},
		{"digits kept", "42 Main St", "42mainst"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeName(tt.input)
			require.Equal(t, tt.want, got, "normalizeName(%q)", tt.input)
		})
	}
}

// --- namesEqual ---

func TestNamesEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"same", "Jane Doe", "Jane Doe", true},
		{"case insensitive lower", "Jane Doe", "jane doe", true},
		{"case insensitive upper", "Jane Doe", "JANE DOE", true},
		{"apostrophe difference", "Jane's Doe", "Jane Doe", false}, // "s" from "Jane's" is kept
		{"different surnames", "Jane Doe", "Jane Smith", false},
		{"one empty", "Jane Doe", "", false},
		{"both empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := namesEqual(tt.a, tt.b)
			require.Equal(t, tt.want, got, "namesEqual(%q, %q)", tt.a, tt.b)
		})
	}
}

func float64PtrEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func fmtF64(p *float64) string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%g", *p)
}

func fmtInt(p *int) string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d", *p)
}
