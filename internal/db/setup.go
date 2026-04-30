package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

var _, b, _, _ = runtime.Caller(0)

// SetupDatabase opens and migrates the application database. It selects the
// backend based on the DATABASE_URL environment variable: a postgres:// or
// postgresql:// URL uses PostgreSQL; anything else uses a local SQLite file.
// SQLite files are stored at /data/biblioteka.db when a /data directory is
// present (Docker/production), or at <project root>/db/biblioteka.db
// otherwise, where the project root is derived by getProjectRoot().
// Migrations are applied automatically before returning.
func SetupDatabase(ctx context.Context) (*DB, error) {
	slog.DebugContext(ctx, "db: Setting up database")

	if databaseURL := os.Getenv("DATABASE_URL"); isPostgresURL(databaseURL) {
		return setupPostgres(ctx, databaseURL)
	}
	return setupSQLite(ctx)
}

func isPostgresURL(url string) bool {
	return strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://")
}

func setupPostgres(ctx context.Context, databaseURL string) (*DB, error) {
	slog.DebugContext(ctx, "db: Opening PostgreSQL database")
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	slog.DebugContext(ctx, "db: PostgreSQL connection established")

	d := &DB{DB: sqlDB, Dialect: DialectPostgres}

	if err := runMigrations(ctx, d); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to run migrations on postgres: %w", err)
	}

	slog.InfoContext(ctx, "PostgreSQL database setup complete")
	return d, nil
}

func setupSQLite(ctx context.Context) (*DB, error) {
	// Determine database path: prefer mounted /data folder, fall back to ./db
	var dbFilePath string
	if _, err := os.Stat("/data"); err == nil {
		dbFilePath = "/data/biblioteka.db"
		slog.DebugContext(ctx, "db: Using mounted /data folder", slog.String(otelkeys.Path, dbFilePath))
	} else {
		dbFilePath = filepath.Join(getProjectRoot(), "db", "biblioteka.db")
		slog.DebugContext(ctx, "db: Using local db folder", slog.String(otelkeys.Path, dbFilePath))
	}

	// Ensure parent directory exists
	dbDir := filepath.Dir(dbFilePath)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		slog.ErrorContext(ctx, "Failed to create database directory",
			slog.String(otelkeys.Path, dbDir),
			slog.Any(otelkeys.Error, err),
		)
		return nil, fmt.Errorf("failed to create database directory %s: %w", dbDir, err)
	}

	// Open database with modernc.org/sqlite pure Go driver
	slog.DebugContext(ctx, "db: Opening database", slog.String(otelkeys.Path, dbFilePath))
	sqlDB, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to open database",
			slog.String(otelkeys.Path, dbFilePath),
			slog.Any(otelkeys.Error, err),
		)
		return nil, fmt.Errorf("failed to open database at %s: %w", dbFilePath, err)
	}

	// Verify connection
	if err := sqlDB.PingContext(ctx); err != nil {
		slog.ErrorContext(ctx, "Failed to ping database",
			slog.String(otelkeys.Path, dbFilePath),
			slog.Any(otelkeys.Error, err),
		)
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database at %s: %w", dbFilePath, err)
	}

	slog.DebugContext(ctx, "db: Database connection established", slog.String(otelkeys.Path, dbFilePath))

	// Pin the pool to a single connection so that all connection-scoped PRAGMAs
	// (foreign_keys, synchronous, temp_store, cache_size) are consistently
	// applied to every query. SQLite serializes writers regardless of pool size,
	// so this adds no meaningful write-latency penalty.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Set PRAGMAs for better performance and integrity.
	//   temp_store = MEMORY: keep sort buffers (temp B-trees) in RAM instead of
	//     writing them to disk temp files. All sort-heavy queries benefit.
	//   cache_size = -16384: 16 MB page cache per connection (default is ~8 MB).
	//     More cached pages means fewer disk reads for hot working sets.
	if _, err := sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
		PRAGMA temp_store = MEMORY;
		PRAGMA cache_size = -16384;
	`); err != nil {
		slog.ErrorContext(ctx, "Failed to set PRAGMAs",
			slog.String(otelkeys.Path, dbFilePath),
			slog.Any(otelkeys.Error, err),
		)
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to set PRAGMAs on database at %s: %w", dbFilePath, err)
	}

	slog.DebugContext(ctx, "db: PRAGMAs set successfully", slog.String(otelkeys.Path, dbFilePath))

	d := &DB{DB: sqlDB, Dialect: DialectSQLite}

	// Run migrations
	if err := runMigrations(ctx, d); err != nil {
		slog.ErrorContext(ctx, "Failed to run migrations",
			slog.String(otelkeys.Path, dbFilePath),
			slog.Any(otelkeys.Error, err),
		)
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to run migrations on database at %s: %w", dbFilePath, err)
	}

	// Verify FTS5 index integrity and auto-rebuild if corrupted. A failed
	// integrity check or rebuild is non-fatal: it is logged and the server
	// continues to start regardless of the outcome. If the FTS index remains
	// corrupted, search requests may return incomplete results or fail until
	// a rebuild succeeds, including via POST /api/admin/search/reindex.
	ftsCtx, ftsCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer ftsCancel()

	if err := d.CheckFTSIntegrity(ftsCtx); err != nil {
		slog.WarnContext(ctx, "FTS5 index integrity check failed, attempting rebuild",
			slog.String(otelkeys.Path, dbFilePath),
			slog.Any(otelkeys.Error, err),
		)
		if rbErr := d.RebuildFTS(ftsCtx); rbErr != nil {
			slog.ErrorContext(ctx, "FTS5 index rebuild failed; search requests may fail or return incomplete results until the index is rebuilt",
				slog.String(otelkeys.Path, dbFilePath),
				slog.Any(otelkeys.Error, rbErr),
			)
		} else {
			slog.InfoContext(ctx, "FTS5 index rebuilt successfully", slog.String(otelkeys.Path, dbFilePath))
		}
	}

	slog.InfoContext(ctx, "Database setup complete", slog.String(otelkeys.Path, dbFilePath))
	return d, nil
}

func getProjectRoot() string {
	return filepath.Join(filepath.Dir(b), "../..") //nolint:gocritic // This is a safe operation.
}
