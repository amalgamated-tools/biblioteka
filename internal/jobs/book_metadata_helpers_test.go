package jobs

import (
	"context"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/exif"
	"github.com/amalgamated-tools/biblioteka/internal/pathparser"
)

// TestDeriveTitle verifies that deriveTitle correctly derives a book title from
// the file name by stripping the extension when it matches the declared file type.
func TestDeriveTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileName string
		fileType string
		path     string
		want     string
	}{
		{
			name:     "extension matches type",
			fileName: "My Book.epub",
			fileType: "epub",
			path:     "/library/My Book.epub",
			want:     "My Book",
		},
		{
			name:     "extension does not match type",
			fileName: "My Book.epub",
			fileType: "pdf",
			path:     "/library/My Book.epub",
			want:     "My Book.epub",
		},
		{
			name:     "case-insensitive extension match",
			fileName: "My Book.EPUB",
			fileType: "epub",
			path:     "/library/My Book.EPUB",
			want:     "My Book",
		},
		{
			name:     "case-insensitive type match",
			fileName: "My Book.epub",
			fileType: "EPUB",
			path:     "/library/My Book.epub",
			want:     "My Book",
		},
		{
			name:     "no extension in filename",
			fileName: "My Book",
			fileType: "epub",
			path:     "/library/My Book",
			want:     "My Book",
		},
		{
			name:     "pdf extension matches",
			fileName: "Technical Manual.pdf",
			fileType: "pdf",
			path:     "/library/Technical Manual.pdf",
			want:     "Technical Manual",
		},
		{
			name:     "mobi extension matches",
			fileName: "Adventure.mobi",
			fileType: "mobi",
			path:     "/library/Adventure.mobi",
			want:     "Adventure",
		},
		{
			name:     "empty filename",
			fileName: "",
			fileType: "epub",
			path:     "/library/",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deriveTitle(context.Background(), tt.fileName, tt.fileType, tt.path)
			if got != tt.want {
				t.Errorf("deriveTitle(%q, %q, %q) = %q, want %q",
					tt.fileName, tt.fileType, tt.path, got, tt.want)
			}
		})
	}
}

// TestResolveAuthorAndTitle verifies author and title resolution from
// ExifToolOutput metadata and path-derived PathInfo.
func TestResolveAuthorAndTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		meta         *exif.ExifToolOutput
		pathInfo     pathparser.PathInfo
		currentTitle string
		wantAuthor   string
		wantTitle    string
	}{
		{
			name:         "nil meta uses currentTitle and pathInfo author",
			meta:         nil,
			pathInfo:     pathparser.PathInfo{Author: "Path Author", Title: ""},
			currentTitle: "Filename Title",
			wantAuthor:   "Path Author",
			wantTitle:    "Filename Title",
		},
		{
			name:         "nil meta uses pathInfo title when present",
			meta:         nil,
			pathInfo:     pathparser.PathInfo{Author: "Path Author", Title: "Path Title"},
			currentTitle: "Filename Title",
			wantAuthor:   "Path Author",
			wantTitle:    "Path Title",
		},
		{
			name:         "meta title overrides currentTitle",
			meta:         &exif.ExifToolOutput{Title: "Meta Title", Author: "Meta Author"},
			pathInfo:     pathparser.PathInfo{Author: "Path Author", Title: "Path Title"},
			currentTitle: "Filename Title",
			wantAuthor:   "Meta Author",
			wantTitle:    "Meta Title",
		},
		{
			name:         "meta title takes precedence over pathInfo title",
			meta:         &exif.ExifToolOutput{Title: "Meta Title"},
			pathInfo:     pathparser.PathInfo{Title: "Path Title"},
			currentTitle: "Filename Title",
			wantAuthor:   "",
			wantTitle:    "Meta Title",
		},
		{
			name:         "blank meta author falls back to pathInfo author",
			meta:         &exif.ExifToolOutput{Title: "Meta Title", Author: ""},
			pathInfo:     pathparser.PathInfo{Author: "Path Author"},
			currentTitle: "Filename Title",
			wantAuthor:   "Path Author",
			wantTitle:    "Meta Title",
		},
		{
			name:         "Unknown meta author falls back to pathInfo author",
			meta:         &exif.ExifToolOutput{Title: "Meta Title", Author: "Unknown"},
			pathInfo:     pathparser.PathInfo{Author: "Path Author"},
			currentTitle: "Filename Title",
			wantAuthor:   "Path Author",
			wantTitle:    "Meta Title",
		},
		{
			name:         "meta author with whitespace is trimmed before comparison",
			meta:         &exif.ExifToolOutput{Author: "  Meta Author  "},
			pathInfo:     pathparser.PathInfo{Author: "Path Author"},
			currentTitle: "Title",
			wantAuthor:   "Meta Author",
			wantTitle:    "Title",
		},
		{
			name:         "whitespace-only meta author falls back to pathInfo author",
			meta:         &exif.ExifToolOutput{Author: "   "},
			pathInfo:     pathparser.PathInfo{Author: "Path Author"},
			currentTitle: "Title",
			wantAuthor:   "Path Author",
			wantTitle:    "Title",
		},
		{
			name:         "empty meta and pathInfo returns currentTitle and empty author",
			meta:         &exif.ExifToolOutput{},
			pathInfo:     pathparser.PathInfo{},
			currentTitle: "Filename Title",
			wantAuthor:   "",
			wantTitle:    "Filename Title",
		},
		{
			name:         "nil meta and empty pathInfo returns currentTitle",
			meta:         nil,
			pathInfo:     pathparser.PathInfo{},
			currentTitle: "Current Title",
			wantAuthor:   "",
			wantTitle:    "Current Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotAuthor, gotTitle := resolveAuthorAndTitle(tt.meta, tt.pathInfo, tt.currentTitle)
			if gotAuthor != tt.wantAuthor {
				t.Errorf("resolveAuthorAndTitle() author = %q, want %q", gotAuthor, tt.wantAuthor)
			}
			if gotTitle != tt.wantTitle {
				t.Errorf("resolveAuthorAndTitle() title = %q, want %q", gotTitle, tt.wantTitle)
			}
		})
	}
}
