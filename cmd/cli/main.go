package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/goodreads"
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

	// If there is an additional argument, or the first argument does not
	// correspond to an existing path, treat it as a subcommand. Otherwise,
	// fall back to the legacy `cli <file>` behavior even if the file name
	// happens to match a reserved command.
	if len(os.Args) >= 3 || !pathExists(cmd) {
		switch cmd {
		case "goodreads-search":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: %s goodreads-search <query>\n", os.Args[0])
				os.Exit(1)
			}
			err = runGoodreadsSearch(ctx, os.Args[2])
		case "goodreads-search-isbn":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: %s goodreads-search-isbn <isbn>\n", os.Args[0])
				os.Exit(1)
			}
			err = runGoodreadsSearchByISBN(ctx, os.Args[2])
		case "goodreads-get-by-asin":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: %s goodreads-get-by-asin <asin>\n", os.Args[0])
				os.Exit(1)
			}
			err = runGoodreadsGetByASIN(ctx, os.Args[2])
		case "goodreads-get-by-id":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: %s goodreads-get-by-id <goodreads-id>\n", os.Args[0])
				os.Exit(1)
			}
			err = runGoodreadsGetByID(ctx, os.Args[2])
		case "goodreads-get-by-legacy-id":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: %s goodreads-get-by-legacy-id <legacy-id>\n", os.Args[0])
				os.Exit(1)
			}
			legacyID, parseErr := strconv.ParseInt(os.Args[2], 10, 64)
			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "Invalid legacy ID: %s\n", os.Args[2])
				os.Exit(1)
			}
			err = runGoodreadsGetByLegacyID(ctx, legacyID)
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
			if len(os.Args) >= 3 {
				fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
				printUsage()
				os.Exit(1)
			}
			// Backwards compatibility: treat first argument as a file path.
			err = runProcessFile(ctx, cmd)
		}
	} else {
		// No additional arguments and the path exists: treat as `cli <file>`.
		err = runProcessFile(ctx, cmd)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func pathExists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [arguments]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "       %s <file>                          (legacy) Process a single book file\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  process-file <file>                    Process a single book file\n")
	fmt.Fprintf(os.Stderr, "  scan-directory <directory> [library-id] Scan a directory and enqueue files for processing\n")
	fmt.Fprintf(os.Stderr, "  goodreads-search <query>              Search Goodreads for a book by query\n")
	fmt.Fprintf(os.Stderr, "  goodreads-search-isbn <isbn>          Search Goodreads for a book by ISBN\n")
	fmt.Fprintf(os.Stderr, "  goodreads-get-by-asin <asin>          Get a book from Goodreads by ASIN\n")
	fmt.Fprintf(os.Stderr, "  goodreads-get-by-id <goodreads-id>    Get a book from Goodreads by ID\n")
	fmt.Fprintf(os.Stderr, "  goodreads-get-by-legacy-id <legacy-id> Get a book from Goodreads by legacy ID\n")
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

	ext, err := metadata.NewExtractor(ctx)
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
		return fmt.Errorf("failed to configure Redis client: %w", err)
	}
	defer func() {
		if cerr := w.Close(); cerr != nil {
			slog.WarnContext(ctx, "failed to close worker", slog.Any(otelkeys.Error, cerr))
		}
	}()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", path, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat directory %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %q is not a directory", absPath)
	}

	payload := jobs.ScanPathPayload{
		Path:        absPath,
		LibraryID:   libraryID,
		LibraryRoot: absPath,
	}

	err = jobs.ScanDirectory(ctx, w, payload)
	if err != nil {
		return fmt.Errorf("error scanning directory %q: %w", absPath, err)
	}

	fmt.Printf("Successfully scanned directory: %s\n", absPath)
	return nil
}

func runGoodreadsSearch(ctx context.Context, query string) error {
	client := goodreads.NewClient()
	results, err := client.Search(ctx, query)
	if err != nil {
		return fmt.Errorf("error searching Goodreads for query %q: %w", query, err)
	}

	if len(results) == 0 {
		fmt.Printf("No results found for query: %s\n", query)
		return nil
	}

	fmt.Printf("Goodreads search results for query: %s\n", query)
	for i, book := range results {
		fmt.Printf("%d. %s by %s (Goodreads ID: %s)\n", i+1, book.BookTitle, book.AuthorName, book.BookID)
	}

	return nil
}

func runGoodreadsSearchByISBN(ctx context.Context, isbn string) error {
	client := goodreads.NewClient()
	results, err := client.SearchByISBN(ctx, isbn)
	if err != nil {
		return fmt.Errorf("error searching Goodreads for ISBN %q: %w", isbn, err)
	}

	if len(results) == 0 {
		fmt.Printf("No results found for ISBN: %s\n", isbn)
		return nil
	}

	fmt.Printf("Goodreads search results for ISBN: %s\n", isbn)
	for i, book := range results {
		fmt.Printf("%d. %s by %s (Goodreads ID: %s)\n", i+1, book.BookTitle, book.AuthorName, book.BookID)
	}

	return nil
}

func runGoodreadsGetByASIN(ctx context.Context, asin string) error {
	client := goodreads.NewClient()
	result, err := client.GetBookByASIN(ctx, asin)
	if err != nil {
		return fmt.Errorf("error searching Goodreads for ASIN %q: %w", asin, err)
	}

	if result == nil {
		fmt.Printf("No results found for ASIN: %s\n", asin)
		return nil
	}
	fmt.Printf("Goodreads search result for ASIN: %s\n", asin)
	fmt.Printf("Title: %s\n", result.BookTitle)
	fmt.Printf("Author: %s\n", result.AuthorName)
	fmt.Printf("Goodreads ID: %s\n", result.BookID)

	return nil
}

func runGoodreadsGetByID(ctx context.Context, id string) error {
	client := goodreads.NewClient()
	result, err := client.GetBookByID(ctx, id)
	if err != nil {
		return fmt.Errorf("error getting book from Goodreads for ID %q: %w", id, err)
	}

	if result == nil {
		fmt.Printf("No results found for Goodreads ID: %s\n", id)
		return nil
	}
	fmt.Printf("Goodreads search result for ID: %s\n", id)
	fmt.Printf("Title: %s\n", result.BookTitle)
	fmt.Printf("Author: %s\n", result.AuthorName)
	fmt.Printf("ASIN: %s\n", result.BookASIN)
	fmt.Printf("Goodreads ID: %s\n", result.BookID)

	return nil
}

func runGoodreadsGetByLegacyID(ctx context.Context, legacyID int64) error {
	client := goodreads.NewClient()
	result, err := client.GetBookByLegacyID(ctx, legacyID)
	if err != nil {
		return fmt.Errorf("error getting book from Goodreads for legacy ID %d: %w", legacyID, err)
	}

	if result == nil {
		fmt.Printf("No results found for Goodreads legacy ID: %d\n", legacyID)
		return nil
	}
	fmt.Printf("Goodreads search result for legacy ID: %d\n", legacyID)
	fmt.Printf("Title: %s\n", result.BookTitle)
	fmt.Printf("Author: %s\n", result.AuthorName)
	fmt.Printf("ASIN: %s\n", result.BookASIN)
	fmt.Printf("Goodreads ID: %s\n", result.BookID)
	return nil
}
