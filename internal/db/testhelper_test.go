package db

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

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

// newTestPostgresDB creates a PostgreSQL database scoped to a unique schema
// for this test. The TEST_DATABASE_URL environment variable must be set to a
// valid PostgreSQL DSN; the test is skipped when it is absent.
//
// A fresh schema is created before the test and dropped (CASCADE) in a
// t.Cleanup callback, so each test starts with an empty, fully-migrated
// database without interfering with other tests.
func newTestPostgresDB(t *testing.T) *DB {
	t.Helper()
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}
	t.Setenv("BIBLIOTEKA_ENV", "test")

	schema := pgSchemaName(t.Name())
	quoted := pgQuoteIdent(schema)
	ctx := t.Context()

	// Use a short-lived admin connection to create the schema.
	adminDB, err := sql.Open("pgx", baseURL)
	require.NoError(t, err, "newTestPostgresDB: open admin connection")
	defer adminDB.Close()
	// Drop any leftover schema from a previous failed run, then create fresh.
	_, err = adminDB.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+quoted+" CASCADE")
	require.NoError(t, err, "newTestPostgresDB: drop schema")
	_, err = adminDB.ExecContext(ctx, "CREATE SCHEMA "+quoted)
	require.NoError(t, err, "newTestPostgresDB: create schema")

	// Drop the schema after the test finishes.
	t.Cleanup(func() {
		cleanDB, err2 := sql.Open("pgx", baseURL)
		if err2 != nil {
			return
		}
		defer cleanDB.Close()
		dropCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = cleanDB.ExecContext(dropCtx, "DROP SCHEMA IF EXISTS "+quoted+" CASCADE")
	})

	// Open a new connection scoped to the isolated schema.
	schemaURL := pgAppendSearchPath(baseURL, schema)
	sqlDB, err := sql.Open("pgx", schemaURL)
	require.NoError(t, err, "newTestPostgresDB: open scoped connection")
	t.Cleanup(func() { _ = sqlDB.Close() })

	d := &DB{DB: sqlDB, Dialect: DialectPostgres}
	err = runMigrations(ctx, d)
	require.NoError(t, err, "newTestPostgresDB: migrations")

	return d
}

// pgSchemaName converts a test name into a PostgreSQL-safe, unquoted schema
// identifier: lowercase letters, digits, and underscores only, prefixed with
// "t_" so it always starts with a letter, and capped at 63 bytes.
func pgSchemaName(testName string) string {
	var b strings.Builder
	b.WriteString("t_")
	for _, r := range strings.ToLower(testName) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	s := b.String()
	if len(s) > 63 {
		s = s[:63]
	}
	return s
}

// pgAppendSearchPath appends a search_path query parameter to a PostgreSQL DSN.
func pgAppendSearchPath(baseURL, schema string) string {
	if strings.Contains(baseURL, "?") {
		return baseURL + "&search_path=" + schema
	}
	return baseURL + "?search_path=" + schema
}

// pgQuoteIdent returns a double-quoted PostgreSQL identifier. Any embedded
// double-quote characters are doubled per the SQL standard (defense in depth —
// pgSchemaName already ensures the identifier contains only [a-z0-9_]).
func pgQuoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
