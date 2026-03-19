package jobs

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"
)

func TestProcessBookFile_NilDatabase(t *testing.T) {
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	err = ProcessBookFile(context.Background(), nil, ext, ProcessFilePayload{
		Path:     "/tmp/test.epub",
		FileName: "test.epub",
		FileType: "epub",
	})
	if err == nil {
		t.Fatal("expected error for nil database")
	}
}

func TestProcessBookFile_NilExtractor(t *testing.T) {
	database := newTestDB(t)

	err := ProcessBookFile(context.Background(), database, nil, ProcessFilePayload{
		Path:     "/tmp/test.epub",
		FileName: "test.epub",
		FileType: "epub",
	})
	if err == nil {
		t.Fatal("expected error for nil extractor")
	}
}

func TestProcessBookFile_EmptyPath(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:     "",
		FileName: "test.epub",
		FileType: "epub",
	})
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestProcessBookFile_WhitespacePath(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:     "   ",
		FileName: "test.epub",
		FileType: "epub",
	})
	if err == nil {
		t.Fatal("expected error for whitespace-only path")
	}
}

func TestProcessBookFile_EmptyFileName(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:     "/tmp/test.epub",
		FileName: "",
		FileType: "epub",
	})
	if err == nil {
		t.Fatal("expected error for empty file name")
	}
}

func TestProcessBookFile_EmptyFileType(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:     "/tmp/test.epub",
		FileName: "test.epub",
		FileType: "",
	})
	if err == nil {
		t.Fatal("expected error for empty file type")
	}
}

func TestProcessBookFile_TitleFromFilename(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	// Use a non-epub file so metadata extraction fails and title comes from filename
	dir := t.TempDir()
	path := filepath.Join(dir, "My Cool Book.pdf")
	// Create an empty file so the path exists (extraction will fail but that's OK)
	if err := os.WriteFile(path, []byte("not a real pdf"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:     path,
		FileName: "My Cool Book.pdf",
		FileType: "pdf",
		FileSize: 14,
	})
	if err != nil {
		t.Fatalf("ProcessBookFile() error: %v", err)
	}

	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	// Extension should be stripped since it matches fileType
	if books[0].Title != "My Cool Book" {
		t.Errorf("expected title %q, got %q", "My Cool Book", books[0].Title)
	}
}

func TestProcessBookFile_TitleFromFilename_NoExtensionMatch(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "noext")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:     path,
		FileName: "noext",
		FileType: "pdf",
		FileSize: 7,
	})
	if err != nil {
		t.Fatalf("ProcessBookFile() error: %v", err)
	}

	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	// No extension to strip, so title is the full filename
	if books[0].Title != "noext" {
		t.Errorf("expected title %q, got %q", "noext", books[0].Title)
	}
}

func TestProcessBookFile_ExistingAuthorReused(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	// Pre-create the author
	_, err = database.CreateAuthor(context.Background(), "F. Scott Fitzgerald", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create author: %v", err)
	}

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "gatsby.epub")
	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:     epubPath,
		FileName: "gatsby.epub",
		FileType: "epub",
		FileSize: 1024,
	})
	if err != nil {
		t.Fatalf("ProcessBookFile() error: %v", err)
	}

	// Should have reused the existing author, not created a duplicate
	authors, err := database.ListAuthors(context.Background())
	if err != nil {
		t.Fatalf("list authors: %v", err)
	}
	if len(authors) != 1 {
		t.Fatalf("expected 1 author (reused), got %d", len(authors))
	}

	// Verify the book is associated with the existing author
	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	bookAuthors, err := database.GetBookAuthors(context.Background(), books[0].ID)
	if err != nil {
		t.Fatalf("get book authors: %v", err)
	}
	if len(bookAuthors) != 1 {
		t.Fatalf("expected 1 book author, got %d", len(bookAuthors))
	}
	if bookAuthors[0].Name != "F. Scott Fitzgerald" {
		t.Errorf("expected author %q, got %q", "F. Scott Fitzgerald", bookAuthors[0].Name)
	}
}

func TestProcessBookFile_NoAuthorInMetadata(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "noauthor.epub")
	// Empty creator means no author in metadata
	testutils.MakeTestEPUB(t, epubPath, "Anonymous Work", "", "some-id-123")

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:     epubPath,
		FileName: "noauthor.epub",
		FileType: "epub",
		FileSize: 512,
	})
	if err != nil {
		t.Fatalf("ProcessBookFile() error: %v", err)
	}

	// No author should be created
	authors, err := database.ListAuthors(context.Background())
	if err != nil {
		t.Fatalf("list authors: %v", err)
	}
	if len(authors) != 0 {
		t.Errorf("expected 0 authors, got %d", len(authors))
	}

	// Book should still be created
	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	if books[0].Title != "Anonymous Work" {
		t.Errorf("expected title %q, got %q", "Anonymous Work", books[0].Title)
	}
}

func TestProcessBookFile_MetadataExtractionFails(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	// Create a file that isn't a valid EPUB — extraction will fail
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.epub")
	if err := os.WriteFile(path, []byte("not a valid epub"), 0o644); err != nil {
		t.Fatalf("write broken.epub: %v", err)
	}

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:     path,
		FileName: "broken.epub",
		FileType: "epub",
		FileSize: 16,
	})
	if err != nil {
		t.Fatalf("ProcessBookFile() error: %v", err)
	}

	// Book should still be created with filename-derived title
	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	if books[0].Title != "broken" {
		t.Errorf("expected title %q, got %q", "broken", books[0].Title)
	}
}

func TestProcessBookFile_ISBN10(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "isbn10.epub")
	testutils.MakeTestEPUB(t, epubPath, "ISBN10 Book", "Author", "isbn:0-306-40615-2")

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:     epubPath,
		FileName: "isbn10.epub",
		FileType: "epub",
		FileSize: 1024,
	})
	if err != nil {
		t.Fatalf("ProcessBookFile() error: %v", err)
	}

	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	if books[0].ISBN10 == nil || *books[0].ISBN10 != "0306406152" {
		t.Errorf("expected ISBN10 %q, got %v", "0306406152", books[0].ISBN10)
	}
	if books[0].ISBN13 != nil {
		t.Errorf("expected ISBN13 nil, got %v", books[0].ISBN13)
	}
}

func TestProcessBookFile_OrganizeFiles(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	// Enable file reorganization.
	if err := database.SetSetting(context.Background(), "organize_files", "true"); err != nil {
		t.Fatalf("set setting: %v", err)
	}

	// Create a library root with Author/Book.epub structure.
	root := t.TempDir()
	epubPath := filepath.Join(root, "The Great Gatsby.epub")
	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:        epubPath,
		FileName:    "The Great Gatsby.epub",
		FileType:    "epub",
		FileSize:    1024,
		LibraryRoot: root,
	})
	if err != nil {
		t.Fatalf("ProcessBookFile() error: %v", err)
	}

	// Verify the original file was moved.
	if _, err := os.Stat(epubPath); !os.IsNotExist(err) {
		t.Error("expected original file to be removed after reorganization")
	}

	// Verify the file was moved to the expected Author/Title/ structure.
	expectedPath := filepath.Join(root, "F. Scott Fitzgerald", "The Great Gatsby", "The Great Gatsby.epub")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected reorganized file at %q, got error: %v", expectedPath, err)
	}

	// Verify book_files.file_path matches the new location.
	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	files, err := database.ListBookFiles(context.Background(), books[0].ID)
	if err != nil {
		t.Fatalf("list book files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].FilePath != expectedPath {
		t.Errorf("expected file path %q, got %q", expectedPath, files[0].FilePath)
	}
}

func TestProcessBookFile_ContinuesFromReorganizedPathWhenSourceMoved(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	root := t.TempDir()
	originalPath := filepath.Join(root, "F. Scott Fitzgerald - The Great Gatsby.epub")
	reorganizedPath := filepath.Join(root, "F. Scott Fitzgerald", "The Great Gatsby", "F. Scott Fitzgerald - The Great Gatsby.epub")

	if err := os.MkdirAll(filepath.Dir(reorganizedPath), 0o755); err != nil {
		t.Fatalf("mkdir reorganized dir: %v", err)
	}
	testutils.MakeTestEPUB(t, reorganizedPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	err = ProcessBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:        originalPath,
		FileName:    filepath.Base(originalPath),
		FileType:    "epub",
		FileSize:    1024,
		LibraryRoot: root,
	})
	if err != nil {
		t.Fatalf("ProcessBookFile() error: %v", err)
	}

	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}

	files, err := database.ListBookFiles(context.Background(), books[0].ID)
	if err != nil {
		t.Fatalf("list book files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].FilePath != reorganizedPath {
		t.Errorf("expected file path %q, got %q", reorganizedPath, files[0].FilePath)
	}
}

func TestResolveSourcePath_ReturnsErrorWhenCandidateLookupFails(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	root := t.TempDir()
	originalPath := filepath.Join(root, "F. Scott Fitzgerald - The Great Gatsby.epub")
	reorganizedPath := filepath.Join(root, "F. Scott Fitzgerald", "The Great Gatsby", "F. Scott Fitzgerald - The Great Gatsby.epub")

	// Create the file at the reorganized location only (source is "missing").
	if err := os.MkdirAll(filepath.Dir(reorganizedPath), 0o755); err != nil {
		t.Fatalf("mkdir reorganized dir: %v", err)
	}
	testutils.MakeTestEPUB(t, reorganizedPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	candidateLookupErr := errors.New("candidate lookup failed")
	lookup := func(ctx context.Context, database *db.DB, path string) (*db.BookFile, error) {
		switch path {
		case originalPath:
			return nil, sql.ErrNoRows
		case reorganizedPath:
			return nil, candidateLookupErr
		default:
			return database.GetBookFileByPath(ctx, path)
		}
	}

	err = processBookFile(context.Background(), database, ext, ProcessFilePayload{
		Path:        originalPath,
		FileName:    filepath.Base(originalPath),
		FileType:    "epub",
		FileSize:    1024,
		LibraryRoot: root,
	}, lookup)
	if err == nil {
		t.Fatalf("expected processBookFile() to return an error when candidate lookup fails due to DB error")
	}
	if !strings.Contains(err.Error(), candidateLookupErr.Error()) {
		t.Fatalf("expected error to include %q, got %v", candidateLookupErr.Error(), err)
	}
}
