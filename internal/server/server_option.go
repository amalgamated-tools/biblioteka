package server

import (
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/worker"
)

// ServerOption is a functional option for configuring a Server.
type ServerOption func(*Server)

// WithAddr sets the listen address (e.g. "0.0.0.0"). Takes precedence over
// WithPort when both are supplied to NewServer.
func WithAddr(addr string) ServerOption {
	return func(s *Server) {
		s.addr = addr
	}
}

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

// WithJWTManager attaches the JWT token manager used for issuing and
// validating user sessions.
func WithJWTManager(jwt *auth.JWTManager) ServerOption {
	return func(s *Server) {
		s.JWT = jwt
	}
}

// WithRequireAuth sets the middleware applied to all authenticated routes.
func WithRequireAuth(requireAuth func(http.Handler) http.Handler) ServerOption {
	return func(s *Server) {
		s.requireAuth = requireAuth
	}
}

// WithAuthRateLimiter attaches the per-IP rate limiter applied to the login
// and OIDC callback endpoints.
func WithAuthRateLimiter(authLimiter *auth.RateLimiter) ServerOption {
	return func(s *Server) {
		s.authLimiter = authLimiter
	}
}

// WithWorker attaches the background job worker so handlers can enqueue tasks.
func WithWorker(w *worker.Worker) ServerOption {
	return func(s *Server) {
		s.Worker = w
	}
}

// WithVersion sets the application version string returned by the /api/version
// endpoint.
func WithVersion(version string) ServerOption {
	return func(s *Server) {
		s.version = version
	}
}
