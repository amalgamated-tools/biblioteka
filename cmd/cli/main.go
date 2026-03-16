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
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file>\n", os.Args[0])
		os.Exit(1)
	}

	ctx := context.Background()
	path := os.Args[1]

	if err := run(ctx, path); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, path string) error {
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
