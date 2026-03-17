package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/worker"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()
	cmd := os.Args[1]

	var err error
	switch cmd {
	case "process-file":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s process-file <file>\n", os.Args[0])
			os.Exit(1)
		}
		err = runProcessFile(ctx, os.Args[2])
	case "scan-directory":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s scan-directory <directory> [library-id]\n", os.Args[0])
			os.Exit(1)
		}
		libraryID := ""
		if len(os.Args) >= 4 {
			libraryID = os.Args[3]
		}
		err = runScanDirectory(ctx, os.Args[2], libraryID)
	default:
		// Backwards compatibility: treat first argument as a file path.
		err = runProcessFile(ctx, cmd)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [arguments]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  process-file <file>                    Process a single book file\n")
	fmt.Fprintf(os.Stderr, "  scan-directory <directory> [library-id] Scan a directory and enqueue files for processing\n")
}

func runProcessFile(ctx context.Context, path string) error {
	fileName := filepath.Base(path)
	fileExt := filepath.Ext(path)
	fileType := ""
	if len(fileExt) > 0 {
		fileType = strings.ToLower(fileExt[1:])
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error stating file %q: %w", path, err)
	}
	fileSize := info.Size()

	database, err := db.SetupDatabase(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to setup database", slog.Any(otelkeys.Error, err))
		return fmt.Errorf("failed to setup database: %w", err)
	}
	defer func() { _ = database.Close() }()

	ext, err := metadata.NewExtractor()
	if err != nil {
		return fmt.Errorf("failed to create metadata extractor: %w", err)
	}
	defer ext.Close()

	err = jobs.ProcessBookFile(
		ctx,
		database,
		ext,
		jobs.ProcessFilePayload{
			FileName: fileName,
			FileType: fileType,
			Path:     path,
			FileSize: fileSize,
		},
	)
	if err != nil {
		return fmt.Errorf("error processing file %q: %w", path, err)
	}

	fmt.Printf("Successfully processed file: %s\n", path)
	return nil
}

func runScanDirectory(ctx context.Context, path string, libraryID string) error {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	w, err := worker.New(redisURL)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	defer func() { _ = w.Close() }()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", path, err)
	}

	err = jobs.ScanDirectory(ctx, w, jobs.ScanPathPayload{
		Path:        absPath,
		LibraryID:   libraryID,
		LibraryRoot: absPath,
	})
	if err != nil {
		return fmt.Errorf("error scanning directory %q: %w", absPath, err)
	}

	fmt.Printf("Successfully scanned directory: %s\n", absPath)
	return nil
}
