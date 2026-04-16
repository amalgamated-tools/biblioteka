package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMigrations_SQLite verifies that every SQLite migration file applies
// cleanly to a fresh database and is recorded in schema_migrations.
func TestMigrations_SQLite(t *testing.T) {
	d := newTestDB(t)

	var applied int
	err := d.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM schema_migrations").Scan(&applied)
	require.NoError(t, err)

	expected := countMigrationFiles(t, "sqlite")
	require.Equal(t, expected, applied, "all SQLite migration files should be recorded in schema_migrations")
}

// TestMigrations_Postgres verifies that every PostgreSQL migration file
// applies cleanly to a fresh database and is recorded in schema_migrations.
func TestMigrations_Postgres(t *testing.T) {
	d := newTestPostgresDB(t)

	var applied int
	err := d.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM schema_migrations").Scan(&applied)
	require.NoError(t, err)

	expected := countMigrationFiles(t, "postgres")
	require.Equal(t, expected, applied, "all PostgreSQL migration files should be recorded in schema_migrations")
}

func countMigrationFiles(t *testing.T, dialect string) int {
	t.Helper()
	dir := filepath.Join(getProjectRoot(), "db", "migrations", dialect)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err, "reading migrations directory")
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			count++
		}
	}
	return count
}
