package sidecar

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateBaseName(t *testing.T) {
	tests := []struct {
		name     string
		baseName string
		wantErr  bool
	}{
		// Valid inputs.
		// Empty string is intentionally allowed: it signals "use the default sidecar
		// filename" (e.g. "cover.jpg" / "metadata.opf") rather than a stem-based name.
		{name: "empty string is allowed", baseName: "", wantErr: false},
		{name: "simple name", baseName: "mybook", wantErr: false},
		{name: "name with extension", baseName: "mybook.epub", wantErr: false},
		{name: "name with spaces", baseName: "my book title", wantErr: false},
		{name: "name with leading dot", baseName: ".hidden", wantErr: false},
		{name: "name with hyphens", baseName: "my-book-title", wantErr: false},
		{name: "name with apostrophe", baseName: "Alice's Adventures", wantErr: false},
		// Invalid inputs.
		{name: "dot", baseName: ".", wantErr: true},
		{name: "double dot", baseName: "..", wantErr: true},
		{name: "forward slash in name", baseName: "sub/file", wantErr: true},
		{name: "backslash in name", baseName: `sub\file`, wantErr: true},
		{name: "leading slash", baseName: "/file", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBaseName(tt.baseName)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid sidecar base name")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSidecarTarget(t *testing.T) {
	tests := []struct {
		name             string
		bookFilePath     string
		organizationType string
		wantDir          string
		wantBaseName     string
		wantErr          bool
	}{
		// Empty path is always an error.
		{
			name:             "empty path",
			bookFilePath:     "",
			organizationType: organizationBookPerFile,
			wantErr:          true,
		},
		// book_per_folder: baseName is always empty.
		{
			name:             "book per folder produces empty baseName",
			bookFilePath:     "/books/mybook.epub",
			organizationType: "book_per_folder",
			wantDir:          "/books",
			wantBaseName:     "",
		},
		// book_per_file: baseName is the file stem.
		{
			name:             "book per file produces file stem",
			bookFilePath:     "/books/mybook.epub",
			organizationType: organizationBookPerFile,
			wantDir:          "/books",
			wantBaseName:     "mybook",
		},
		{
			name:             "book per file with no extension",
			bookFilePath:     "/books/mybook",
			organizationType: organizationBookPerFile,
			wantDir:          "/books",
			wantBaseName:     "mybook",
		},
		{
			name:             "book per file with stem-less filename falls back to full filename",
			bookFilePath:     "/books/.epub",
			organizationType: organizationBookPerFile,
			wantDir:          "/books",
			wantBaseName:     ".epub",
		},
		// Unknown organization type: behaves like book_per_folder (empty baseName).
		{
			name:             "unknown organization type produces empty baseName",
			bookFilePath:     "/books/mybook.epub",
			organizationType: "unknown_type",
			wantDir:          "/books",
			wantBaseName:     "",
		},
		// Nested path.
		{
			name:             "nested path book per file",
			bookFilePath:     "/a/b/c/title.mobi",
			organizationType: organizationBookPerFile,
			wantDir:          "/a/b/c",
			wantBaseName:     "title",
		},
		// Paths that produce an invalid baseName should return an error.
		{
			name:             "dot-only path with book per file produces error",
			bookFilePath:     ".",
			organizationType: organizationBookPerFile,
			wantErr:          true,
		},
		{
			name:             "double-dot path with book per file produces error",
			bookFilePath:     "..",
			organizationType: organizationBookPerFile,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, baseName, err := sidecarTarget(tt.bookFilePath, tt.organizationType)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantDir, dir)
			require.Equal(t, tt.wantBaseName, baseName)
		})
	}
}
