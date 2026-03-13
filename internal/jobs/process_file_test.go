package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
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

	if err := db.RunMigrations(sqlDB, db.DialectSQLite); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("migrations: %v", err)
	}

	t.Cleanup(func() { _ = sqlDB.Close() })
	return &db.DB{DB: sqlDB, Dialect: db.DialectSQLite}
}

func TestProcessFileHandler(t *testing.T) {
	database := newTestDB(t)
	handler := NewProcessFileHandler(database)

	payload, err := json.Marshal(ProcessFilePayload{
		Path:     "/books/My Book.epub",
		FileName: "My Book.epub",
		FileType: "epub",
		FileSize: 1024,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler: %v", err)
	}

	books, err := database.ListBooks()
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	if books[0].Title != "My Book" {
		t.Errorf("expected title %q, got %q", "My Book", books[0].Title)
	}

	files, err := database.ListBookFiles(books[0].ID)
	if err != nil {
		t.Fatalf("list book files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].FileType != "epub" {
		t.Errorf("expected file type %q, got %q", "epub", files[0].FileType)
	}
	if files[0].FilePath != "/books/My Book.epub" {
		t.Errorf("expected file path %q, got %q", "/books/My Book.epub", files[0].FilePath)
	}
	if files[0].FileSize != 1024 {
		t.Errorf("expected file size 1024, got %d", files[0].FileSize)
	}
}

func TestProcessFileHandler_EmptyPath(t *testing.T) {
	database := newTestDB(t)
	handler := NewProcessFileHandler(database)

	payload, _ := json.Marshal(ProcessFilePayload{Path: ""})
	err := handler(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}
