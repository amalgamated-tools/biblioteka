package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
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
	extractor, err := metadata.NewExtractor()
	if err != nil {
		t.Fatalf("failed to create metadata extractor: %v", err)
	}
	defer extractor.Close()
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
}

func TestProcessFileHandler_EmptyPath(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor()
	if err != nil {
		t.Fatalf("failed to create metadata extractor: %v", err)
	}
	defer extractor.Close()
	handler := NewProcessFileHandler(database, extractor)

	payload, _ := json.Marshal(ProcessFilePayload{Path: ""})
	err = handler(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}
