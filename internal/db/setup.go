package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

var (
	_, b, _, _ = runtime.Caller(0)
)

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

	if err := runMigrations(d); err != nil {
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
		slog.DebugContext(ctx, "Using mounted /data folder", slog.String("path", dbFilePath))
	} else {
		dbFilePath = filepath.Join(getProjectRoot(), "db", "biblioteka.db")
		slog.DebugContext(ctx, "Using local db folder", slog.String("path", dbFilePath))
	}

	// Ensure parent directory exists
	dbDir := filepath.Dir(dbFilePath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		slog.ErrorContext(ctx, "Failed to create database directory", slog.String("path", dbDir), slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("failed to create database directory %s: %w", dbDir, err)
	}

	// Open database with modernc.org/sqlite pure Go driver
	slog.DebugContext(ctx, "Opening database", slog.String("path", dbFilePath))
	sqlDB, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to open database", slog.String("path", dbFilePath), slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("failed to open database at %s: %w", dbFilePath, err)
	}

	// Verify connection
	if err := sqlDB.PingContext(ctx); err != nil {
		slog.ErrorContext(ctx, "Failed to ping database", slog.String("path", dbFilePath), slog.Any(otelkeys.Error, err))
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database at %s: %w", dbFilePath, err)
	}

	slog.DebugContext(ctx, "Database connection established", slog.String("path", dbFilePath))

	// Set PRAGMAs for better performance and integrity
	if _, err := sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`); err != nil {
		slog.ErrorContext(ctx, "Failed to set PRAGMAs", slog.String("path", dbFilePath), slog.Any(otelkeys.Error, err))
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to set PRAGMAs on database at %s: %w", dbFilePath, err)
	}

	slog.DebugContext(ctx, "PRAGMAs set successfully", slog.String("path", dbFilePath))

	d := &DB{DB: sqlDB, Dialect: DialectSQLite}

	// Run migrations
	if err := runMigrations(d); err != nil {
		slog.ErrorContext(ctx, "Failed to run migrations", slog.String("path", dbFilePath), slog.Any(otelkeys.Error, err))
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to run migrations on database at %s: %w", dbFilePath, err)
	}

	slog.InfoContext(ctx, "Database setup complete", slog.String("path", dbFilePath))
	return d, nil
}

// RunMigrations runs all pending database migrations on the given connection.
// It is exported so that packages outside db (e.g. handler tests) can set up
// a fully-migrated in-memory database without duplicating the schema inline.
func RunMigrations(sqlDB *sql.DB, dialect Dialect) error {
	d := &DB{DB: sqlDB, Dialect: dialect}
	return runMigrations(d)
}

// runMigrations reads and executes all SQL migration files
// Supports dbmate format with '-- migrate:up' and '-- migrate:down' markers
func runMigrations(d *DB) error {
	ctx := context.Background()

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
			slog.DebugContext(ctx, "Migration already applied", slog.String("version", version))
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
		slog.InfoContext(ctx, "Migration applied", slog.String("version", version))
	}

	return nil
}

// extractUpSQL extracts the SQL between '-- migrate:up' and '-- migrate:down' markers
func extractUpSQL(content string) string {
	lines := strings.Split(content, "\n")
	var upLines []string
	inUpBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "-- migrate:up" {
			inUpBlock = true
			continue
		}

		if trimmed == "-- migrate:down" {
			break
		}

		if inUpBlock && trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			upLines = append(upLines, line)
		}
	}

	return strings.TrimSpace(strings.Join(upLines, "\n"))
}

// splitStatements splits SQL by semicolon, handling strings and inline comments properly
func splitStatements(sql string) []string {
	// First, remove inline comments (-- to end of line)
	sql = removeInlineComments(sql)

	var statements []string
	var current strings.Builder
	inString := false
	var stringChar rune
	inCreateStatement := false
	inTriggerStatement := false
	triggerBodyDepth := 0
	triggerBodyClosed := false
	var i int
	runes := []rune(sql)

	for i < len(runes) {
		char := runes[i]

		if !inString && (char == '\'' || char == '"') {
			inString = true
			stringChar = char
			current.WriteRune(char)
		} else if !inString && isSQLIdentifierRune(char) {
			start := i
			for i < len(runes) && isSQLIdentifierRune(runes[i]) {
				i++
			}

			word := strings.ToUpper(string(runes[start:i]))
			current.WriteString(string(runes[start:i]))

			if !inTriggerStatement {
				if word == "CREATE" {
					inCreateStatement = true
				} else if inCreateStatement && word == "TRIGGER" {
					inTriggerStatement = true
					triggerBodyDepth = 0
					triggerBodyClosed = false
				}
			}

			if inTriggerStatement {
				switch word {
				case "BEGIN":
					triggerBodyDepth++
				case "END":
					if triggerBodyDepth > 0 {
						triggerBodyDepth--
					}
					if triggerBodyDepth == 0 {
						triggerBodyClosed = true
					}
				}
			}

			continue
		} else if inString && char == stringChar {
			if i+1 < len(runes) && runes[i+1] == stringChar {
				// Escaped quote
				current.WriteRune(char)
				current.WriteRune(char)
				i++
			} else {
				inString = false
				current.WriteRune(char)
			}
		} else if !inString && char == ';' {
			if inTriggerStatement && !triggerBodyClosed {
				current.WriteRune(char)
			} else {
				statements = append(statements, current.String())
				current.Reset()
				inCreateStatement = false
				inTriggerStatement = false
				triggerBodyDepth = 0
				triggerBodyClosed = false
			}
		} else {
			current.WriteRune(char)
		}

		i++
	}

	// Add any remaining statement
	if current.Len() > 0 {
		statements = append(statements, current.String())
	}

	return statements
}

func isSQLIdentifierRune(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_'
}

// removeInlineComments removes SQL inline comments (-- to end of line) while preserving strings
func removeInlineComments(sql string) string {
	var result strings.Builder
	runes := []rune(sql)
	inString := false
	var stringChar rune

	for i := 0; i < len(runes); i++ {
		char := runes[i]

		// Handle string delimiters
		if !inString && (char == '\'' || char == '"') {
			inString = true
			stringChar = char
			result.WriteRune(char)
		} else if inString && char == stringChar {
			if i+1 < len(runes) && runes[i+1] == stringChar {
				// Escaped quote
				result.WriteRune(char)
				result.WriteRune(runes[i+1])
				i++
			} else {
				inString = false
				result.WriteRune(char)
			}
		} else if !inString && char == '-' && i+1 < len(runes) && runes[i+1] == '-' {
			// Found inline comment, skip until end of line (but don't skip the newline itself)
			i += 2
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
			// Don't increment i here, so the newline will be written in the next iteration
			i--
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}

func getProjectRoot() string {
	return filepath.Join(filepath.Dir(b), "../..") //nolint:gocritic // This is a safe operation.
}
