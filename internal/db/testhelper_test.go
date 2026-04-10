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
	t.Cleanup(func() { _ = sqlDB.Close() })

	err = sqlDB.Ping()
	require.NoError(t, err, "newTestDB: ping")

	_, err = sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`)
	require.NoError(t, err, "newTestDB: pragmas")

	d := &DB{DB: sqlDB, Dialect: DialectSQLite}

	err = runMigrations(t.Context(), d)
	require.NoError(t, err, "newTestDB: migrations")

	return d
}
