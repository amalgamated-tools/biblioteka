package pathparser

import (
	"fmt"
	"testing"
)

func TestParseBookPath(t *testing.T) {
	root := "/library"
	tests := []struct {
		name     string
		filePath string
		want     PathInfo
	}{
		{
			name:     "3-level: author/series/numbered file with author and year",
			filePath: "/library/Alexander McCall Smith/No. 1 Ladies' Detective Agency/10. Tea Time for the Traditionally Built - Alexander McCall Smith (2009).epub",
			want: PathInfo{
				Author:         "Alexander McCall Smith",
				Title:          "Tea Time for the Traditionally Built",
				SeriesName:     "No. 1 Ladies' Detective Agency",
				SeriesPosition: pf(10),
				Year:           pi(2009),
			},
		},
		{
			name:     "3-level: second book in series",
			filePath: "/library/Alexander McCall Smith/No. 1 Ladies' Detective Agency/11. The Double Comfort Safari Club - Alexander McCall Smith (2010).epub",
			want: PathInfo{
				Author:         "Alexander McCall Smith",
				Title:          "The Double Comfort Safari Club",
				SeriesName:     "No. 1 Ladies' Detective Agency",
				SeriesPosition: pf(11),
				Year:           pi(2010),
			},
		},
		{
			name:     "2-level: author folder with numbered file",
			filePath: "/library/Agatha Christie/1. The Seven Dials Mystery - Agatha Christie (2010).epub",
			want: PathInfo{
				Author:         "Agatha Christie",
				Title:          "The Seven Dials Mystery",
				SeriesPosition: pf(1),
				Year:           pi(2010),
			},
		},
		{
			name:     "flat file: author dash title",
			filePath: "/library/Mary Shelley - Frankenstein.epub",
			want: PathInfo{
				Author: "Mary Shelley",
				Title:  "Frankenstein",
			},
		},
		{
			name:     "flat file: author dash title (ambiguous)",
			filePath: "/library/Moby Dick - Herman Melville.epub",
			want: PathInfo{
				Author: "Moby Dick",
				Title:  "Herman Melville",
			},
		},
		{
			name:     "1-level: author-title folder with matching filename",
			filePath: "/library/Emily Brontë - Wuthering Heights/Emily Brontë - Wuthering Heights.epub",
			want: PathInfo{
				Author: "Emily Brontë",
				Title:  "Wuthering Heights",
			},
		},
		{
			name:     "flat file: no dash separator",
			filePath: "/library/Frankenstein.epub",
			want: PathInfo{
				Title: "Frankenstein",
			},
		},
		{
			name:     "2-level: simple author/title",
			filePath: "/library/Jane Austen/Pride and Prejudice.epub",
			want: PathInfo{
				Author: "Jane Austen",
				Title:  "Pride and Prejudice",
			},
		},
		{
			name:     "2-level: filename with year only",
			filePath: "/library/Jane Austen/Pride and Prejudice (1813).epub",
			want: PathInfo{
				Author: "Jane Austen",
				Title:  "Pride and Prejudice",
				Year:   pi(1813),
			},
		},
		{
			name:     "Calibre-style: Author/Title/file.epub does not create phantom series",
			filePath: "/library/Jane Austen/Pride and Prejudice/book.epub",
			want: PathInfo{
				Author: "Jane Austen",
				Title:  "book",
			},
		},
		{
			name:     "3-level: series dir only used when filename has position prefix",
			filePath: "/library/Brandon Sanderson/Mistborn/2. The Well of Ascension.epub",
			want: PathInfo{
				Author:         "Brandon Sanderson",
				Title:          "The Well of Ascension",
				SeriesName:     "Mistborn",
				SeriesPosition: pf(2),
			},
		},
		{
			name:     "trailing author with hyphenated name is stripped",
			filePath: "/library/Tea Time for the Traditionally Built - Jean-Paul Sartre.epub",
			want: PathInfo{
				Author: "Tea Time for the Traditionally Built",
				Title:  "Jean-Paul Sartre",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseBookPath(tt.filePath, root)
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

func pf(f float64) *float64 { return &f }
func pi(i int) *int         { return &i }

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
