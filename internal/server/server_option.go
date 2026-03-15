package server

import (
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/worker"
)

type ServerOption func(*Server)

func WithAddr(addr string) ServerOption {
	return func(s *Server) {
		s.addr = addr
	}
}

func WithPort(port int) ServerOption {
	return func(s *Server) {
		s.port = port
	}
}

func WithDB(db *db.DB) ServerOption {
	return func(s *Server) {
		s.DB = db
	}
}

func WithJWTManager(jwt *auth.JWTManager) ServerOption {
	return func(s *Server) {
		s.JWT = jwt
	}
}

func WithRequireAuth(requireAuth func(http.Handler) http.Handler) ServerOption {
	return func(s *Server) {
		s.requireAuth = requireAuth
	}
}

func WithAuthRateLimiter(authLimiter *auth.RateLimiter) ServerOption {
	return func(s *Server) {
		s.authLimiter = authLimiter
	}
}

func WithWorker(w *worker.Worker) ServerOption {
	return func(s *Server) {
		s.Worker = w
	}
}

func WithVersion(version string) ServerOption {
	return func(s *Server) {
		s.version = version
	}
}
