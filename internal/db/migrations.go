package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// RunMigrations runs all pending database migrations on the given connection.
// It is exported so that packages outside db (e.g. handler tests) can set up
// a fully-migrated in-memory database without duplicating the schema inline.
func RunMigrations(ctx context.Context, sqlDB *sql.DB, dialect Dialect) error {
	d := &DB{DB: sqlDB, Dialect: dialect}
	return runMigrations(ctx, d)
}

// runMigrations reads and executes all SQL migration files
// Supports dbmate format with '-- migrate:up' and '-- migrate:down' markers
func runMigrations(ctx context.Context, d *DB) error {
	// Create migrations table if it doesn't exist
	if _, err := d.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Select migration directory based on dialect
	subdir := "sqlite"
	if d.Dialect == DialectPostgres {
		subdir = "postgres"
	}
	migrationsDir := filepath.Join(getProjectRoot(), "db", "migrations", subdir)
	slog.InfoContext(ctx, "Running database migrations", slog.String("migrationsDir", migrationsDir))
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory %s: %w", migrationsDir, err)
	}

	var migrations []fs.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrations = append(migrations, entry)
		}
	}

	// Sort migrations by filename (timestamp-based)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name() < migrations[j].Name()
	})

	// Execute each migration
	for _, migration := range migrations {
		filename := migration.Name()
		version := strings.TrimSuffix(filename, ".sql")

		// Check if migration has already been applied
		var applied int
		err := d.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&applied)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if applied > 0 {
			slog.DebugContext(ctx, "Migration already applied", slog.String(otelkeys.Version, version))
			continue
		}

		// Read migration file
		migrationPath := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Extract the "up" SQL from the migration file (dbmate format)
		upSQL := extractUpSQL(string(content))
		if upSQL == "" {
			return fmt.Errorf("migration %s has no '-- migrate:up' section", filename)
		}

		// Run this migration in a transaction to ensure all-or-nothing execution
		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", filename, err)
		}

		// Execute migration - split by semicolon to handle multiple statements
		statements := splitStatements(upSQL)
		for _, stmt := range statements {
			if strings.TrimSpace(stmt) == "" {
				continue
			}

			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				// Any error means the migration did not fully apply; roll back.
				if rbErr := tx.Rollback(); rbErr != nil {
					return fmt.Errorf("failed to execute migration %s: %v (rollback error: %w)", filename, err, rbErr)
				}
				return fmt.Errorf("failed to execute migration %s: %w", filename, err)
			}
		}

		// Record migration within the same transaction
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("failed to record migration %s: %v (rollback error: %w)", filename, err, rbErr)
			}
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", filename, err)
		}
		if env := os.Getenv("BIBLIOTEKA_ENV"); env != "test" {
			slog.InfoContext(ctx, "Migration applied", slog.String(otelkeys.Version, version))
		}
	}

	if err := backfillKoboTokenHashes(ctx, d); err != nil {
		return fmt.Errorf("failed to backfill kobo token hashes: %w", err)
	}

	return nil
}
