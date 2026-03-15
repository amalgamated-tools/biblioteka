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

func (s *Server) setupFrontend(ctx context.Context) {
	// Serve the embedded frontend SPA
	frontendFS, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		slog.ErrorContext(ctx, "failed to setup frontend filesystem", slog.Any(otelkeys.Error, err))
		panic(fmt.Sprintf("failed to setup frontend filesystem: %v", err))
	}

	fileServer := http.FileServer(http.FS(frontendFS))

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
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
