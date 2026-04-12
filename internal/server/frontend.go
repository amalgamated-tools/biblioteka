package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// staticCacheMiddleware wraps a file server and sets Cache-Control headers
// appropriate for the asset type:
//   - Vite content-hashed assets under /assets/ get a 1-year immutable cache,
//     eliminating network round-trips for repeat visitors.
//   - index.html gets no-cache so the browser always revalidates the entry
//     point and picks up new asset hashes when the app is updated.
//   - All other static files are served without a Cache-Control override and
//     rely on http.FileServer's default ETag/Last-Modified behaviour.
func staticCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/assets/"):
			// Content-hashed filenames never change; cache indefinitely.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case r.URL.Path == "/" || r.URL.Path == "/index.html":
			// Entry point must be revalidated so browsers pick up new hashes.
			w.Header().Set("Cache-Control", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setupFrontend(ctx context.Context) {
	// Serve the embedded frontend SPA
	frontendFS, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		slog.ErrorContext(ctx, "failed to setup frontend filesystem", slog.Any(otelkeys.Error, err))
		panic(fmt.Sprintf("failed to setup frontend filesystem: %v", err))
	}

	fileServer := staticCacheMiddleware(http.FileServer(http.FS(frontendFS)))

	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If the request is for an API route that wasn't matched, return 404
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the file directly
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if the file exists in the embedded filesystem
		f, err := frontendFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found — serve index.html for SPA client-side routing
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}
