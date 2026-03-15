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

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

var _, b, _, _ = runtime.Caller(0)

func SetupDatabase(ctx context.Context) (*DB, error) {
	slog.DebugContext(ctx, "Setting up database")

	if databaseURL := os.Getenv("DATABASE_URL"); isPostgresURL(databaseURL) {
		return setupPostgres(ctx, databaseURL)
	}
	return setupSQLite(ctx)
}

func isPostgresURL(url string) bool {
	return strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://")
}

func setupPostgres(ctx context.Context, databaseURL string) (*DB, error) {
	slog.DebugContext(ctx, "Opening PostgreSQL database")
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	slog.DebugContext(ctx, "PostgreSQL connection established")

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
		slog.DebugContext(ctx, "Using mounted /data folder", slog.String(otelkeys.Path, dbFilePath))
	} else {
		dbFilePath = filepath.Join(getProjectRoot(), "db", "biblioteka.db")
		slog.DebugContext(ctx, "Using local db folder", slog.String(otelkeys.Path, dbFilePath))
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
	slog.DebugContext(ctx, "Opening database", slog.String(otelkeys.Path, dbFilePath))
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

	slog.DebugContext(ctx, "Database connection established", slog.String(otelkeys.Path, dbFilePath))

	// Set PRAGMAs for better performance and integrity
	if _, err := sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`); err != nil {
		slog.ErrorContext(ctx, "Failed to set PRAGMAs",
			slog.String(otelkeys.Path, dbFilePath),
			slog.Any(otelkeys.Error, err),
		)
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to set PRAGMAs on database at %s: %w", dbFilePath, err)
	}

	slog.DebugContext(ctx, "PRAGMAs set successfully", slog.String(otelkeys.Path, dbFilePath))

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

	slog.InfoContext(ctx, "Database setup complete", slog.String(otelkeys.Path, dbFilePath))
	return d, nil
}

func getProjectRoot() string {
	return filepath.Join(filepath.Dir(b), "../..") //nolint:gocritic // This is a safe operation.
}
