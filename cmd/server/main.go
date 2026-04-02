package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/otel"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/server"
	"github.com/amalgamated-tools/biblioteka/internal/worker"
	"golang.org/x/sync/errgroup"
)

var version = "dev"

//	@title						Biblioteka API
//	@version					1.0
//	@description				Personal library management API
//	@BasePath					/api
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Enter "Bearer {token}"
//
//	@securityDefinitions.apikey	KOSyncUser
//	@in							header
//	@name						x-auth-user
//	@description				KOSync username
//
//	@securityDefinitions.apikey	KOSyncKey
//	@in							header
//	@name						x-auth-key
//	@description				KOSync authentication key (hex-encoded MD5 of password)

func main() {
	cancelCtx, cancelAll := context.WithCancel(context.Background())
	otel.SetupLogger(cancelCtx)
	slog.InfoContext(cancelCtx, "biblioteka", slog.String(otelkeys.Version, version))
	if err := realMain(cancelCtx); err != nil {
		slog.ErrorContext(cancelCtx, "error occurred", slog.Any(otelkeys.Error, err))
		cancelAll()
	}
}

// This is the real main function. That's why it's called realMain.
func realMain(cancelCtx context.Context) error { //nolint:contextcheck // The newctx context comes from the StartTracer function, so it's already wrapped.
	port := flag.Int("port", 8080, "port to listen on")
	mode := flag.String("mode", "all", `run mode: "all" runs the HTTP server and the background worker together (default), "server" runs only the HTTP server, "worker" runs only the background worker`)
	flag.Parse()

	if *mode != "all" && *mode != "server" && *mode != "worker" {
		return fmt.Errorf("invalid mode %q: must be one of all, server, worker", *mode)
	}

	// Allow PORT env var to override the flag default
	if p := os.Getenv("PORT"); p != "" {
		_, err := fmt.Sscanf(p, "%d", port)
		if err != nil {
			slog.ErrorContext(cancelCtx, "invalid PORT value", slog.Any(otelkeys.Error, err))
			return fmt.Errorf("invalid PORT value: %w", err)
		}
	}
	database, err := db.SetupDatabase(cancelCtx)
	if err != nil {
		slog.ErrorContext(cancelCtx, "failed to setup database", slog.Any(otelkeys.Error, err))
		return fmt.Errorf("failed to setup database: %w", err)
	}
	defer func() { _ = database.Close() }()

	// Set up the background worker (always needed: server mode uses it for enqueuing, worker/all modes also process jobs)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	w, err := worker.New(redisURL)
	if err != nil {
		slog.ErrorContext(cancelCtx, "failed to setup worker", slog.Any(otelkeys.Error, err))
		return fmt.Errorf("failed to setup worker: %w", err)
	}
	defer func() { _ = w.Close() }()

	runServer := *mode == "all" || *mode == "server"
	runWorker := *mode == "all" || *mode == "worker"

	// Register job handlers and schedules only when this instance processes jobs
	if runWorker {
		// Set up the metadata Extractor, which reads metadata from uploaded files (using ExifTool when available).
		extractor, err := metadata.NewExtractor(cancelCtx)
		if err != nil {
			slog.ErrorContext(cancelCtx, "failed to setup metadata extractor", slog.Any(otelkeys.Error, err))
			return fmt.Errorf("failed to setup metadata extractor: %w", err)
		}
		defer extractor.Close(cancelCtx)

		w.Register(cancelCtx, jobs.JobScanPath, jobs.NewScanPathHandler(w))
		w.Register(cancelCtx, jobs.JobProcessFile, jobs.NewProcessFileHandler(database, extractor))
		w.Register(cancelCtx, jobs.JobScanLibrary, jobs.NewScanLibraryHandler(w))
		w.Register(cancelCtx, jobs.JobScanLibraries, jobs.NewScanLibrariesHandler(database, w))

		if _, err := w.RegisterSchedule("@every 24h", jobs.JobScanLibraries, struct{}{}); err != nil {
			slog.ErrorContext(cancelCtx, "failed to schedule scan:libraries job", slog.Any(otelkeys.Error, err))
			return fmt.Errorf("failed to schedule scan:libraries job: %w", err)
		}
	}

	if !runServer && !runWorker {
		return fmt.Errorf("mode %q starts neither server nor worker", *mode)
	}

	g, ctx := errgroup.WithContext(cancelCtx)

	if runServer {
		httpServer, err := server.NewServer(
			cancelCtx,
			server.WithPort(*port),
			server.WithDB(database),
			server.WithWorker(w),
			server.WithVersion(version),
		)
		if err != nil {
			slog.ErrorContext(cancelCtx, "failed to create server", slog.Any(otelkeys.Error, err))
			return fmt.Errorf("failed to create server: %w", err)
		}

		g.Go(func() error {
			slog.InfoContext(ctx, "Starting HTTP server", slog.Int(otelkeys.Port, *port))
			return httpServer.Run(ctx)
		})
	}

	if runWorker {
		g.Go(func() error {
			slog.InfoContext(ctx, "Starting background worker")
			if err := w.Start(ctx); err != nil {
				slog.ErrorContext(ctx, "background worker failed", slog.Any(otelkeys.Error, err))
				return fmt.Errorf("background worker failed: %w", err)
			}
			slog.InfoContext(ctx, "Background worker stopped")
			return nil
		})
	}

	return g.Wait()
}
