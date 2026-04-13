package handlers

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"
)

// newTestDB creates an in-memory SQLite database with all real migrations applied.
func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err, "newTestDB: open")
	t.Cleanup(func() { _ = sqlDB.Close() })

	// Restrict to one connection: in-memory SQLite databases are per-connection,
	// so concurrent goroutines acquiring different connections would each see an
	// empty database. A single connection serialises all access to the same DB.
	sqlDB.SetMaxOpenConns(1)

	_, err = sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`)
	require.NoError(t, err, "newTestDB: pragmas")

	err = db.RunMigrations(t.Context(), sqlDB, db.DialectSQLite)
	require.NoError(t, err, "newTestDB: migrations")

	return &db.DB{DB: sqlDB, Dialect: db.DialectSQLite}
}

func newTestJWT(t *testing.T) *auth.JWTManager {
	t.Helper()
	jm, err := auth.NewJWTManager("testsecret", time.Hour)
	require.NoError(t, err, "newTestJWT")
	return jm
}

// withUserID returns a copy of r with the user ID injected into the context.
func withUserID(r *http.Request, userID string) *http.Request {
	ctx := auth.ContextWithUserID(r.Context(), userID)
	return r.WithContext(ctx)
}
