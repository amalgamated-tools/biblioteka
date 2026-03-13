package db

import (
	"database/sql"
	"testing"
)

// newTestDB creates an in-memory SQLite database with all migrations applied.
// It registers a cleanup function so the database is closed when the test ends.
func newTestDB(t *testing.T) *DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("newTestDB: open: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("newTestDB: ping: %v", err)
	}

	if _, err := sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("newTestDB: pragmas: %v", err)
	}

	d := &DB{DB: sqlDB, Dialect: DialectSQLite}

	if err := runMigrations(d); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("newTestDB: migrations: %v", err)
	}

	t.Cleanup(func() { _ = sqlDB.Close() })
	return d
}
