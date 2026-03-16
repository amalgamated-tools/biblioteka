package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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
	fileExt := filepath.Ext(path)

	database, err := db.SetupDatabase(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to setup database", slog.Any(otelkeys.Error, err))
		os.Exit(1)
	}
	defer func() { _ = database.Close() }()

	ext, err := metadata.NewExtractor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer ext.Close()

	err = jobs.ProcessBookFile(
		ctx,
		database,
		ext,
		jobs.ProcessFilePayload{
			FileName: path,
			FileType: fileExt[1:], // Replace with actual file type if known
			Path:     path,
			FileSize: 0, // Replace with actual file size if known},
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error processing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully processed file: %s\n", path)
}
