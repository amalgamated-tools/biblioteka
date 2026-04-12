package server

import (
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/worker"
)

// ServerOption is a functional option for configuring a Server.
type ServerOption func(*Server)

// WithPort sets the TCP port the server listens on.
func WithPort(port int) ServerOption {
	return func(s *Server) {
		s.port = port
	}
}

// WithDB attaches a database connection to the server.
func WithDB(db *db.DB) ServerOption {
	return func(s *Server) {
		s.DB = db
	}
}

// WithWorker attaches the background job worker so handlers can enqueue tasks.
func WithWorker(w *worker.Worker) ServerOption {
	return func(s *Server) {
		s.Worker = w
	}
}

// WithCORSAllowedOrigins configures the list of origins that are permitted to
// make cross-origin requests to the book upload and capture endpoints (e.g.
// browser extension origins such as "moz-extension://abc123"). When the slice
// is empty (the default) no CORS headers are emitted and all cross-origin
// browser requests remain blocked.
func WithCORSAllowedOrigins(origins []string) ServerOption {
	return func(s *Server) {
		s.corsAllowedOrigins = origins
	}
}

// WithVersion sets the application version string returned by the /api/version
// endpoint.
func WithVersion(version string) ServerOption {
	return func(s *Server) {
		s.version = version
	}
}
