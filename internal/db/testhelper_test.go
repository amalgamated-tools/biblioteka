package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// newTestDB creates an in-memory SQLite database with all migrations applied.
// It registers a cleanup function so the database is closed when the test ends.
func newTestDB(t *testing.T) *DB {
	t.Helper()
	t.Setenv("BIBLIOTEKA_ENV", "test")
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err, "newTestDB: open")

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		require.NoError(t, err, "newTestDB: ping")
	}

	if _, err := sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`); err != nil {
		_ = sqlDB.Close()
		require.NoError(t, err, "newTestDB: pragmas")
	}

	d := &DB{DB: sqlDB, Dialect: DialectSQLite}

	if err := runMigrations(t.Context(), d); err != nil {
		_ = sqlDB.Close()
		require.NoError(t, err, "newTestDB: migrations")
	}

	t.Cleanup(func() { _ = sqlDB.Close() })
	return d
}
