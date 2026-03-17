package main

import (
	"context"
	"flag"
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
	ctx := context.Background()

	file := flag.String("file", "", "path to the file to process")
	directory := flag.String("directory", "", "path to the directory to process")
	flag.Parse()
	if *file == "" && *directory == "" {
		fmt.Fprintln(os.Stderr, "error: either --file or --directory must be provided")
		os.Exit(1)
	}
	if *file != "" && *directory != "" {
		fmt.Fprintln(os.Stderr, "error: only one of --file or --directory can be provided")
		os.Exit(1)
	}
	// if a directory is provided, we will call runDirectory, which will call run for each file in the directory
	if *directory != "" {
		if err := runDirectory(ctx, *directory); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := run(ctx, *file); err != nil {
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

func runDirectory(ctx context.Context, dirPath string) error {
	return nil
}
