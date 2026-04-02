package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"
	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *db.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if _, err := sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("pragmas: %v", err)
	}

	if err := db.RunMigrations(t.Context(), sqlDB, db.DialectSQLite); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("migrations: %v", err)
	}

	t.Cleanup(func() { _ = sqlDB.Close() })
	return &db.DB{DB: sqlDB, Dialect: db.DialectSQLite}
}

func TestProcessFileHandler(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("failed to create metadata extractor: %v", err)
	}
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")

	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	payload, err := json.Marshal(ProcessFilePayload{
		Path:     epubPath,
		FileName: filepath.Base(epubPath),
		FileType: "epub",
		FileSize: 1024,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler: %v", err)
	}

	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	if books[0].Title != "The Great Gatsby" {
		t.Errorf("expected title %q, got %q", "The Great Gatsby", books[0].Title)
	}

	files, err := database.ListBookFiles(context.Background(), books[0].ID)
	if err != nil {
		t.Fatalf("list book files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].FileType != "epub" {
		t.Errorf("expected file type %q, got %q", "epub", files[0].FileType)
	}
	if files[0].FilePath != epubPath {
		t.Errorf("expected file path %q, got %q", epubPath, files[0].FilePath)
	}
	if files[0].FileSize != 1024 {
		t.Errorf("expected file size 1024, got %d", files[0].FileSize)
	}

	// Verify author was created and associated with the book
	authors, err := database.GetBookAuthors(context.Background(), books[0].ID)
	if err != nil {
		t.Fatalf("get book authors: %v", err)
	}
	if len(authors) != 1 {
		t.Fatalf("expected 1 author, got %d", len(authors))
	}
	if authors[0].Name != "F. Scott Fitzgerald" {
		t.Errorf("expected author %q, got %q", "F. Scott Fitzgerald", authors[0].Name)
	}

	// Verify ISBN13 was extracted and normalized
	if books[0].ISBN13 == nil || *books[0].ISBN13 != "9780743273565" {
		t.Errorf("expected ISBN13 %q, got %v", "9780743273565", books[0].ISBN13)
	}

	// Verify language was extracted from EPUB metadata
	if books[0].Language == nil || *books[0].Language != "en" {
		t.Errorf("expected language %q, got %v", "en", books[0].Language)
	}
}

func TestProcessFileHandler_MetadataFields(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("failed to create metadata extractor: %v", err)
	}
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "rich.epub")

	testutils.MakeTestEPUBWithOptions(t, epubPath, "Dune", "Frank Herbert", "urn:isbn:9780441172719", testutils.EPUBOptions{
		Description:     "A science fiction masterpiece",
		Publisher:       "Chilton Books",
		PublicationDate: "1965-08-01",
	})

	payload, err := json.Marshal(ProcessFilePayload{
		Path:     epubPath,
		FileName: filepath.Base(epubPath),
		FileType: "epub",
		FileSize: 2048,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler: %v", err)
	}

	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	book := books[0]

	if book.Title != "Dune" {
		t.Errorf("expected title %q, got %q", "Dune", book.Title)
	}
	if book.Description == nil || *book.Description != "A science fiction masterpiece" {
		t.Errorf("expected description %q, got %v", "A science fiction masterpiece", book.Description)
	}
	if book.Publisher == nil || *book.Publisher != "Chilton Books" {
		t.Errorf("expected publisher %q, got %v", "Chilton Books", book.Publisher)
	}
	if book.PublicationDate == nil || *book.PublicationDate != "1965-08-01" {
		t.Errorf("expected publication_date %q, got %v", "1965-08-01", book.PublicationDate)
	}
	if book.Language == nil || *book.Language != "en" {
		t.Errorf("expected language %q, got %v", "en", book.Language)
	}
	if book.ISBN13 == nil || *book.ISBN13 != "9780441172719" {
		t.Errorf("expected ISBN13 %q, got %v", "9780441172719", book.ISBN13)
	}

	// Verify author creation and association
	authors, err := database.GetBookAuthors(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("get book authors: %v", err)
	}
	if len(authors) != 1 {
		t.Fatalf("expected 1 author, got %d", len(authors))
	}
	if authors[0].Name != "Frank Herbert" {
		t.Errorf("expected author %q, got %q", "Frank Herbert", authors[0].Name)
	}
}

func TestProcessFileHandler_EmptyPath(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("failed to create metadata extractor: %v", err)
	}
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)

	payload, err := json.Marshal(ProcessFilePayload{Path: ""})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	err = handler(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestProcessFileHandler_AuthorAndLibraryLinking(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("failed to create metadata extractor: %v", err)
	}
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)

	// Create a library to link the book to.
	lib, err := database.CreateLibrary(context.Background(), "Fiction", `["/books"]`, db.LibraryOrganizationBookPerFolder, true)
	if err != nil {
		t.Fatalf("create library: %v", err)
	}

	// Set up: Author/Title.epub structure in a temp dir simulating a library root.
	root := t.TempDir()
	authorDir := filepath.Join(root, "F. Scott Fitzgerald")
	if err := os.MkdirAll(authorDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	epubPath := filepath.Join(authorDir, "The Great Gatsby.epub")
	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	payload, err := json.Marshal(ProcessFilePayload{
		Path:        epubPath,
		FileName:    filepath.Base(epubPath),
		FileType:    "epub",
		FileSize:    1024,
		LibraryID:   lib.ID,
		LibraryRoot: root,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler: %v", err)
	}

	// Verify book was created.
	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}

	// Verify author was linked.
	authors, err := database.GetBookAuthors(context.Background(), books[0].ID)
	if err != nil {
		t.Fatalf("get book authors: %v", err)
	}
	if len(authors) != 1 {
		t.Fatalf("expected 1 author, got %d", len(authors))
	}
	if authors[0].Name != "F. Scott Fitzgerald" {
		t.Errorf("expected author %q, got %q", "F. Scott Fitzgerald", authors[0].Name)
	}

	// Verify book was added to library.
	libBooks, err := database.ListBooksByLibrary(context.Background(), lib.ID)
	if err != nil {
		t.Fatalf("list library books: %v", err)
	}
	if len(libBooks) != 1 {
		t.Fatalf("expected 1 library book, got %d", len(libBooks))
	}
}

func TestProcessFileHandler_SeriesFromPath(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("failed to create metadata extractor: %v", err)
	}
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)

	// Set up: Author/Series/N. Title.epub
	root := t.TempDir()
	seriesDir := filepath.Join(root, "Alexander McCall Smith", "No. 1 Ladies' Detective Agency")
	if err := os.MkdirAll(seriesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	epubPath := filepath.Join(seriesDir, "10. Tea Time for the Traditionally Built (2009).epub")
	testutils.MakeTestEPUB(t, epubPath, "", "", "")

	payload, err := json.Marshal(ProcessFilePayload{
		Path:        epubPath,
		FileName:    filepath.Base(epubPath),
		FileType:    "epub",
		FileSize:    512,
		LibraryRoot: root,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler: %v", err)
	}

	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}

	// Verify series was linked.
	seriesEntries, err := database.GetBookSeries(context.Background(), books[0].ID)
	if err != nil {
		t.Fatalf("get book series: %v", err)
	}
	if len(seriesEntries) != 1 {
		t.Fatalf("expected 1 series entry, got %d", len(seriesEntries))
	}
	if seriesEntries[0].Series.Name != "No. 1 Ladies' Detective Agency" {
		t.Errorf("expected series %q, got %q", "No. 1 Ladies' Detective Agency", seriesEntries[0].Series.Name)
	}
	if seriesEntries[0].Position == nil || *seriesEntries[0].Position != 10 {
		t.Errorf("expected series position 10, got %v", seriesEntries[0].Position)
	}

	// Verify author was linked from directory.
	authors, err := database.GetBookAuthors(context.Background(), books[0].ID)
	if err != nil {
		t.Fatalf("get book authors: %v", err)
	}
	if len(authors) != 1 {
		t.Fatalf("expected 1 author, got %d", len(authors))
	}
	if authors[0].Name != "Alexander McCall Smith" {
		t.Errorf("expected author %q, got %q", "Alexander McCall Smith", authors[0].Name)
	}
}

func TestProcessFileHandler_DuplicateSkipped(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	if err != nil {
		t.Fatalf("failed to create metadata extractor: %v", err)
	}
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	testutils.MakeTestEPUB(t, epubPath, "Duplicate Book", "Author", "")

	payload, err := json.Marshal(ProcessFilePayload{
		Path:     epubPath,
		FileName: filepath.Base(epubPath),
		FileType: "epub",
		FileSize: 1024,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Process once.
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("first handler call: %v", err)
	}

	// Process again — should be skipped (no error, no duplicate book).
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("second handler call: %v", err)
	}

	books, err := database.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Errorf("expected 1 book after duplicate processing, got %d", len(books))
	}
}
