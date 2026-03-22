package pathparser

import (
	"fmt"
	"path/filepath"
	"testing"
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
				SeriesPosition: pf(10),
				Year:           new(2009),
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
				SeriesPosition: pf(11),
				Year:           new(2010),
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
				SeriesPosition: pf(1),
				Year:           new(2010),
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
				Year:   new(1813),
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
				SeriesPosition: pf(2),
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
			if got.Author != tt.want.Author {
				t.Errorf("Author: got %q, want %q", got.Author, tt.want.Author)
			}
			if got.Title != tt.want.Title {
				t.Errorf("Title: got %q, want %q", got.Title, tt.want.Title)
			}
			if got.SeriesName != tt.want.SeriesName {
				t.Errorf("SeriesName: got %q, want %q", got.SeriesName, tt.want.SeriesName)
			}
			if !float64PtrEqual(got.SeriesPosition, tt.want.SeriesPosition) {
				t.Errorf("SeriesPosition: got %s, want %s", fmtF64(got.SeriesPosition), fmtF64(tt.want.SeriesPosition))
			}
			if !intPtrEqual(got.Year, tt.want.Year) {
				t.Errorf("Year: got %s, want %s", fmtInt(got.Year), fmtInt(tt.want.Year))
			}
		})
	}
}

//go:fix inline
func pf(f float64) *float64 { return new(f) }

//go:fix inline
func pi(i int) *int { return new(i) }

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
