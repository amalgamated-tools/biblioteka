package jobs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/pathparser"

	"github.com/stretchr/testify/require"
)

// TestValidateField verifies that validateField rejects blank and
// whitespace-only values and accepts non-empty ones.
func TestValidateField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fieldName string
		value     string
		wantErr   bool
	}{
		{name: "non-empty value", fieldName: "path", value: "/library/book.epub", wantErr: false},
		{name: "empty value", fieldName: "path", value: "", wantErr: true},
		{name: "whitespace-only value", fieldName: "path", value: "   ", wantErr: true},
		{name: "tab-only value", fieldName: "path", value: "\t", wantErr: true},
		{name: "single character", fieldName: "path", value: "a", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateField(context.Background(), tt.fieldName, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateField(%q, %q) error = %v, wantErr %v",
					tt.fieldName, tt.value, err, tt.wantErr)
			}
		})
	}
}

// TestValidateField_ErrorContainsFieldName verifies that the error message
// includes the field name for debugging.
func TestValidateField_ErrorContainsFieldName(t *testing.T) {
	t.Parallel()

	err := validateField(context.Background(), "my_special_field", "")
	require.Error(t, err, "expected error, got nil")
	errStr := err.Error()
	if !strings.Contains(errStr, "my_special_field") {
		t.Errorf("error message %q should contain field name %q", errStr, "my_special_field")
	}
}

// TestValidatePayload verifies that validatePayload correctly validates all
// three required payload fields (path, file name, file type).
func TestValidatePayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload ProcessFilePayload
		wantErr bool
	}{
		{
			name:    "all fields valid",
			payload: ProcessFilePayload{Path: "/lib/book.epub", FileName: "book.epub", FileType: "epub"},
			wantErr: false,
		},
		{
			name:    "empty path",
			payload: ProcessFilePayload{Path: "", FileName: "book.epub", FileType: "epub"},
			wantErr: true,
		},
		{
			name:    "whitespace path",
			payload: ProcessFilePayload{Path: "   ", FileName: "book.epub", FileType: "epub"},
			wantErr: true,
		},
		{
			name:    "empty file name",
			payload: ProcessFilePayload{Path: "/lib/book.epub", FileName: "", FileType: "epub"},
			wantErr: true,
		},
		{
			name:    "whitespace file name",
			payload: ProcessFilePayload{Path: "/lib/book.epub", FileName: "  ", FileType: "epub"},
			wantErr: true,
		},
		{
			name:    "empty file type",
			payload: ProcessFilePayload{Path: "/lib/book.epub", FileName: "book.epub", FileType: ""},
			wantErr: true,
		},
		{
			name:    "whitespace file type",
			payload: ProcessFilePayload{Path: "/lib/book.epub", FileName: "book.epub", FileType: "\t"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validatePayload(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestReorganizedCandidatePaths_BookPerFolder verifies that BookPerFolder
// returns an author/title organized path when both author and title are available.
func TestReorganizedCandidatePaths_BookPerFolder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := ProcessFilePayload{
		Path:        filepath.Join(dir, "book.epub"),
		LibraryRoot: dir,
	}
	pathInfo := pathparser.PathInfo{
		Author: "Terry Pratchett",
		Title:  "Guards! Guards!",
	}

	candidates := reorganizedCandidatePaths(context.Background(), p, pathInfo, db.LibraryOrganizationBookPerFolder)

	if len(candidates) != 1 {
		require.Failf(t, "failed", "expected 1 candidate, got %d: %v", len(candidates), candidates)
	}
	// The candidate should be under libraryRoot/Author/Title/
	rel, err := filepath.Rel(dir, candidates[0])
	require.NoError(t, err, "filepath.Rel")
	if rel == "" || rel == "." {
		t.Errorf("expected candidate path to be a subdirectory, got %q", rel)
	}
}

// TestReorganizedCandidatePaths_BookPerFolder_MissingAuthor verifies that
// BookPerFolder returns no candidates when the author is blank.
func TestReorganizedCandidatePaths_BookPerFolder_MissingAuthor(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := ProcessFilePayload{
		Path:        filepath.Join(dir, "book.epub"),
		LibraryRoot: dir,
	}
	pathInfo := pathparser.PathInfo{
		Author: "",
		Title:  "Some Title",
	}

	candidates := reorganizedCandidatePaths(context.Background(), p, pathInfo, db.LibraryOrganizationBookPerFolder)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates, got %d: %v", len(candidates), candidates)
	}
}

// TestReorganizedCandidatePaths_BookPerFolder_MissingTitle verifies that
// BookPerFolder returns no candidates when the title is blank.
func TestReorganizedCandidatePaths_BookPerFolder_MissingTitle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := ProcessFilePayload{
		Path:        filepath.Join(dir, "book.epub"),
		LibraryRoot: dir,
	}
	pathInfo := pathparser.PathInfo{
		Author: "Some Author",
		Title:  "",
	}

	candidates := reorganizedCandidatePaths(context.Background(), p, pathInfo, db.LibraryOrganizationBookPerFolder)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates, got %d: %v", len(candidates), candidates)
	}
}

// TestReorganizedCandidatePaths_BookPerFile verifies that BookPerFile returns a
// flat author-level path when the author is available.
func TestReorganizedCandidatePaths_BookPerFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := ProcessFilePayload{
		Path:        filepath.Join(dir, "book.epub"),
		LibraryRoot: dir,
	}
	pathInfo := pathparser.PathInfo{
		Author: "Terry Pratchett",
	}

	candidates := reorganizedCandidatePaths(context.Background(), p, pathInfo, db.LibraryOrganizationBookPerFile)

	if len(candidates) != 1 {
		require.Failf(t, "failed", "expected 1 candidate, got %d: %v", len(candidates), candidates)
	}
	// Flat layout: libraryRoot/Author/filename
	expected := filepath.Join(dir, "Terry Pratchett", "book.epub")
	if candidates[0] != expected {
		t.Errorf("expected candidate %q, got %q", expected, candidates[0])
	}
}

// TestReorganizedCandidatePaths_BookPerFile_MissingAuthor verifies that
// BookPerFile returns no candidates when the author is blank.
func TestReorganizedCandidatePaths_BookPerFile_MissingAuthor(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := ProcessFilePayload{
		Path:        filepath.Join(dir, "book.epub"),
		LibraryRoot: dir,
	}
	pathInfo := pathparser.PathInfo{Author: ""}

	candidates := reorganizedCandidatePaths(context.Background(), p, pathInfo, db.LibraryOrganizationBookPerFile)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates, got %d: %v", len(candidates), candidates)
	}
}

// TestReorganizedCandidatePaths_NoneOrganization verifies that unknown or
// empty organization types return no candidates.
func TestReorganizedCandidatePaths_NoneOrganization(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := ProcessFilePayload{
		Path:        filepath.Join(dir, "book.epub"),
		LibraryRoot: dir,
	}
	pathInfo := pathparser.PathInfo{
		Author: "Some Author",
		Title:  "Some Title",
	}

	for _, orgType := range []string{"", db.LibraryOrganizationNone, "unknown_type"} {
		t.Run("org="+orgType, func(t *testing.T) {
			t.Parallel()
			candidates := reorganizedCandidatePaths(context.Background(), p, pathInfo, orgType)
			if len(candidates) != 0 {
				t.Errorf("expected 0 candidates for org %q, got %d: %v", orgType, len(candidates), candidates)
			}
		})
	}
}

// TestReorganizedCandidatePaths_EmptyLibraryRoot verifies that an empty
// library root results in no candidates regardless of organization type.
func TestReorganizedCandidatePaths_EmptyLibraryRoot(t *testing.T) {
	t.Parallel()

	p := ProcessFilePayload{
		Path:        "/library/book.epub",
		LibraryRoot: "",
	}
	pathInfo := pathparser.PathInfo{
		Author: "Some Author",
		Title:  "Some Title",
	}

	for _, orgType := range []string{db.LibraryOrganizationBookPerFolder, db.LibraryOrganizationBookPerFile} {
		t.Run("org="+orgType, func(t *testing.T) {
			t.Parallel()
			// TargetPath/TargetPathFlat return "" for empty libraryRoot;
			// reorganizedCandidatePaths must not add empty strings.
			candidates := reorganizedCandidatePaths(context.Background(), p, pathInfo, orgType)
			for _, c := range candidates {
				if c == "" {
					t.Errorf("got empty string candidate for org %q", orgType)
				}
			}
		})
	}
}

// TestReorganizedCandidatePaths_NoDuplicates verifies that the same path is
// not included twice even when different strategies would generate it.
func TestReorganizedCandidatePaths_NoDuplicates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Place the file already at the organized location so both the original
	// and reorganized path happen to be the same.
	author := "Author Name"
	title := "Book Title"
	organizedDir := filepath.Join(dir, author, title)
	if err := os.MkdirAll(organizedDir, 0o750); err != nil {
		require.NoError(t, err, "mkdir")
	}
	filePath := filepath.Join(organizedDir, "book.epub")

	p := ProcessFilePayload{
		Path:        filePath,
		LibraryRoot: dir,
	}
	pathInfo := pathparser.PathInfo{
		Author: author,
		Title:  title,
	}

	candidates := reorganizedCandidatePaths(context.Background(), p, pathInfo, db.LibraryOrganizationBookPerFolder)

	seen := make(map[string]bool)
	for _, c := range candidates {
		if seen[c] {
			t.Errorf("duplicate candidate path %q", c)
		}
		seen[c] = true
	}
}
