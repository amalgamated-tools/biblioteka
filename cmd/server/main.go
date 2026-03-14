package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/otel"
	"github.com/amalgamated-tools/biblioteka/internal/server"
	"github.com/amalgamated-tools/biblioteka/internal/worker"
	"golang.org/x/sync/errgroup"
)

var version = "dev"

func main() {
	otel.SetupLogger()
	slog.Info("biblioteka", slog.String("version", version))
	cancelCtx, cancelAll := context.WithCancel(context.Background())

	if err := realMain(cancelCtx); err != nil {
		slog.ErrorContext(cancelCtx, "error occurred", slog.Any("error", err))
		cancelAll()
	}
}

// This is the real main function. That's why it's called realMain.
func realMain(cancelCtx context.Context) error { //nolint:contextcheck // The newctx context comes from the StartTracer function, so it's already wrapped.
	port := flag.Int("port", 8080, "port to listen on")
	flag.Parse()

	// Allow PORT env var to override the flag default
	if p := os.Getenv("PORT"); p != "" {
		_, err := fmt.Sscanf(p, "%d", port)
		if err != nil {
			slog.ErrorContext(cancelCtx, "invalid PORT value", slog.Any("error", err))
			return fmt.Errorf("invalid PORT value: %w", err)
		}
	}
	database, err := db.SetupDatabase()
	if err != nil {
		slog.ErrorContext(cancelCtx, "failed to setup database", slog.Any("error", err))
		return fmt.Errorf("failed to setup database: %w", err)
	}
	defer func() { _ = database.Close() }()

	// Start the background worker
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	w, err := worker.New(redisURL)
	if err != nil {
		slog.ErrorContext(cancelCtx, "failed to setup worker", slog.Any("error", err))
		return fmt.Errorf("failed to setup worker: %w", err)
	}
	defer func() { _ = w.Close() }()

	// Register background jobs
	w.Register(jobs.JobScanPath, jobs.NewScanPathHandler(w))
	w.Register(jobs.JobProcessFile, jobs.NewProcessFileHandler(database))
	w.Register(jobs.JobScanLibrary, jobs.NewScanLibraryHandler(w))
	w.Register(jobs.JobScanLibraries, jobs.NewScanLibrariesHandler(database, w))

	// Schedule periodic jobs
	if _, err := w.RegisterSchedule("@every 24h", jobs.JobScanLibraries, struct{}{}); err != nil {
		slog.ErrorContext(cancelCtx, "failed to schedule scan:libraries job", slog.Any("error", err))
		return fmt.Errorf("failed to schedule scan:libraries job: %w", err)
	}

	http, err := server.NewServer(
		cancelCtx,
		server.WithPort(*port),
		server.WithDB(database),
		server.WithWorker(w),
	)
	if err != nil {
		slog.ErrorContext(cancelCtx, "failed to create server", slog.Any("error", err))
		return fmt.Errorf("failed to create server: %w", err)
	}

	g, ctx := errgroup.WithContext(cancelCtx)

	g.Go(func() error {
		slog.InfoContext(ctx, "Starting HTTP server", slog.Int("port", *port))
		return http.Run(ctx)
	})

	g.Go(func() error {
		slog.InfoContext(ctx, "Starting background worker")
		w.Start(ctx)
		slog.InfoContext(ctx, "Background worker stopped")
		return nil
	})

	return g.Wait()
}
