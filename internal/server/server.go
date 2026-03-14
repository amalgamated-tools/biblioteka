package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/handlers"
	"github.com/amalgamated-tools/biblioteka/internal/handlers/middleware"
	"github.com/amalgamated-tools/biblioteka/internal/otel"
	"github.com/amalgamated-tools/biblioteka/internal/worker"

	"github.com/justinas/alice"
	"golang.org/x/sync/errgroup"
)

//go:embed dist/*
var embeddedFiles embed.FS

const (
	// UserAgentHeader is the header name for the user agent.
	UserAgentHeader = "User-Agent"
	// HTTPWriteTimeout is the maximum duration before timing out writes of the response.
	HTTPWriteTimeout = 10 * time.Second
	// HTTPReadTimeout is the maximum duration for reading the entire request, including the body.
	HTTPReadTimeout = 10 * time.Second
	// HTTPIdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled.
	HTTPIdleTimeout = 30 * time.Second
	// HTTPRequestTimeout is the maximum duration for handling a single HTTP request.
	HTTPRequestTimeout = 10 * time.Second
	// ShutdownGracePeriod is the time we allow for graceful shutdown of the http server
	// Should be longer than HTTPWriteTimeout, but shorter than the k8s terminationGracePeriodSeconds (30 seconds)
	ShutdownGracePeriod = 15 * time.Second
)

// ShutdownFunc is a function that takes a context and returns an error
type ShutdownFunc func(context.Context) error

// Server represents the HTTP server with embedded frontend
type Server struct {
	addr string

	Address string
	port    int

	DB  *db.DB
	JWT *auth.JWTManager

	Worker *worker.Worker

	oidcHandler     *handlers.OIDCHandler
	authHandler     *handlers.AuthHandler
	configHandler   *handlers.ConfigHandler
	adminHandler    *handlers.AdminHandler
	libraryHandler  *handlers.LibraryHandler
	authorHandler   *handlers.AuthorHandler
	seriesHandler   *handlers.SeriesHandler
	bookHandler     *handlers.BookHandler
	bookFileHandler *handlers.BookFileHandler
	requireAuth     func(http.Handler) http.Handler
	authLimiter     *auth.RateLimiter
	mux             *http.ServeMux
	httpServer      *http.Server
	shutdownFuncs   []ShutdownFunc
}

// NewServer creates a new server instance
func NewServer(ctx context.Context, opts ...ServerOption) (*Server, error) {
	s := &Server{
		mux: http.NewServeMux(),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.port == 0 {
		s.port = 8080
	}
	s.Address = net.JoinHostPort("0.0.0.0", strconv.Itoa(s.port))

	if s.DB == nil {
		database, err := db.SetupDatabase()
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
		s.DB = database
		s.shutdownFuncs = append(s.shutdownFuncs, func(ctx context.Context) error {
			return s.DB.Close()
		})
	}

	if s.JWT == nil {
		jwtSecret := os.Getenv("JWT_SECRET")
		jwtManager, err := auth.NewJWTManager(jwtSecret, 24*time.Hour)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JWT manager: %w", err)
		}
		s.JWT = jwtManager

		if jwtSecret == "" {
			slog.InfoContext(ctx, "WARNING: JWT_SECRET not set, using random secret. Existing JWT tokens will become invalid after a server restart; all users will need to log in again.")
		}
	}

	if s.requireAuth == nil {
		s.requireAuth = auth.Middleware(s.JWT)
	}

	if s.authLimiter == nil {
		s.authLimiter = auth.NewRateLimiter(5, 10)
	}

	// Determine cookie security mode: secure by default, can be disabled for local dev
	secureCookies := os.Getenv("SECURE_COOKIES") != "false"

	s.authHandler = &handlers.AuthHandler{DB: s.DB, JWT: s.JWT}
	s.adminHandler = &handlers.AdminHandler{DB: s.DB}
	s.libraryHandler = &handlers.LibraryHandler{DB: s.DB}
	s.authorHandler = &handlers.AuthorHandler{DB: s.DB}
	s.seriesHandler = &handlers.SeriesHandler{DB: s.DB}
	s.bookHandler = &handlers.BookHandler{DB: s.DB}
	s.bookFileHandler = &handlers.BookFileHandler{DB: s.DB}
	s.configHandler = &handlers.ConfigHandler{
		DB:               s.DB,
		IsOIDCConfigured: func() bool { return s.oidcHandler != nil },
		OnOIDCConfigSet: func(ctx context.Context, issuerURL, clientID, clientSecret, redirectURI string) error {
			oidcHandler, err := handlers.NewOIDCHandler(ctx, s.DB, s.JWT, issuerURL, clientID, clientSecret, redirectURI, secureCookies)
			if err != nil {
				return err
			}
			s.oidcHandler = oidcHandler
			return nil
		},
	}

	// Initialize OIDC handler if configured: env var first, then DB setting
	if issuer := os.Getenv("OIDC_ISSUER_URL"); issuer != "" {
		clientID := os.Getenv("OIDC_CLIENT_ID")
		clientSecret := os.Getenv("OIDC_CLIENT_SECRET")
		redirectURI := os.Getenv("OIDC_REDIRECT_URI")
		if clientID == "" || clientSecret == "" || redirectURI == "" {
			return nil, fmt.Errorf("OIDC_ISSUER_URL is set but one or more of OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, or OIDC_REDIRECT_URI is missing")
		}
		oidcHandler, err := handlers.NewOIDCHandler(ctx, s.DB, s.JWT, issuer, clientID, clientSecret, redirectURI, secureCookies)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
		}
		s.oidcHandler = oidcHandler
		slog.Info("OIDC authentication enabled", slog.String("issuer", issuer))
	} else if dbIssuer, err := s.DB.GetSetting("oidc_issuer_url"); err == nil && dbIssuer != "" {
		dbClientID, _ := s.DB.GetSetting("oidc_client_id")
		dbClientSecret, _ := s.DB.GetSetting("oidc_client_secret")
		dbRedirectURI, _ := s.DB.GetSetting("oidc_redirect_uri")
		if dbClientID != "" && dbClientSecret != "" && dbRedirectURI != "" {
			oidcHandler, err := handlers.NewOIDCHandler(ctx, s.DB, s.JWT, dbIssuer, dbClientID, dbClientSecret, dbRedirectURI, secureCookies)
			if err != nil {
				slog.Warn("failed to initialize OIDC from saved settings", slog.Any("error", err))
			} else {
				s.oidcHandler = oidcHandler
				slog.Info("OIDC authentication enabled from saved settings", slog.String("issuer", dbIssuer))
			}
		}
	}

	s.setupRoutes()
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	newctx, span := otel.StartTracer(ctx, "server.Run")
	defer span.End()
	slog.Debug("Running server", slog.String("address", s.Address))
	ctx, cancel := context.WithCancel(newctx)

	chain := alice.New(
		middleware.RequestIDHandler,
		otel.TraceMiddleware,
		middleware.LoggingMiddleware,
	).Then(s.mux)

	s.httpServer = &http.Server{
		Addr:         s.Address,
		Handler:      chain,
		WriteTimeout: HTTPWriteTimeout,
		ReadTimeout:  HTTPReadTimeout,
		IdleTimeout:  HTTPIdleTimeout,
	}

	go func() {
		err := s.httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", slog.Any("error", err))
			s.shutdownFuncs = append(s.shutdownFuncs, func(_ context.Context) error {
				return err
			})
			cancel()
			return
		}
	}()

	s.shutdownFuncs = append(s.shutdownFuncs, s.httpServer.Shutdown)

	<-ctx.Done()
	return s.shutdown(ctx)
}

func (s *Server) shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, ShutdownGracePeriod)
	defer cancel()

	shutdownGroup, ctx := errgroup.WithContext(ctx)

	for _, shutdownFn := range s.shutdownFuncs {
		fn := shutdownFn
		shutdownGroup.Go(func() error {
			return fn(ctx)
		})
	}

	return shutdownGroup.Wait()
}

func (s *Server) setupRoutes() {
	// Public auth routes (rate-limited)
	s.mux.HandleFunc("/api/auth/signup", s.authLimiter.Limit(s.authHandler.Signup))
	s.mux.HandleFunc("/api/auth/login", s.authLimiter.Limit(s.authHandler.Login))

	// OIDC auth routes — always registered, check handler at request time
	s.mux.HandleFunc("/api/auth/oidc/enabled", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if s.oidcHandler != nil {
			_, _ = fmt.Fprint(w, `{"enabled":true}`)
		} else {
			_, _ = fmt.Fprint(w, `{"enabled":false}`)
		}
	})
	s.mux.HandleFunc("/api/auth/oidc/login", s.authLimiter.Limit(s.oidcRoute((*handlers.OIDCHandler).Login)))
	s.mux.HandleFunc("/api/auth/oidc/callback", s.authLimiter.Limit(s.oidcRoute((*handlers.OIDCHandler).Callback)))
	s.mux.HandleFunc("/api/auth/oidc/link", s.authLimiter.Limit(s.oidcRoute((*handlers.OIDCHandler).Link)))
	s.mux.Handle("/api/auth/oidc/link-nonce", s.requireAuth(http.HandlerFunc(s.oidcRoute((*handlers.OIDCHandler).CreateLinkNonce))))

	// Protected auth routes
	s.mux.Handle("/api/auth/me", s.requireAuth(http.HandlerFunc(s.authHandler.Me)))
	s.mux.Handle("/api/auth/password", s.requireAuth(http.HandlerFunc(s.authHandler.ChangePassword)))

	// Protected config routes
	s.mux.Handle("/api/config/status", s.requireAuth(http.HandlerFunc(s.configHandler.HandleConfigStatus)))
	s.mux.Handle("/api/config/oidc", s.requireAuth(http.HandlerFunc(s.configHandler.HandleOIDCConfig)))

	// Protected admin routes
	s.mux.Handle("/api/admin/users", s.requireAuth(http.HandlerFunc(s.adminHandler.HandleListUsers)))
	s.mux.Handle("/api/admin/users/", s.requireAuth(http.HandlerFunc(s.adminHandler.HandleSetAdmin)))

	// Protected library routes
	s.mux.Handle("/api/libraries", s.requireAuth(http.HandlerFunc(s.libraryHandler.HandleLibraries)))
	s.mux.Handle("/api/libraries/", s.requireAuth(http.HandlerFunc(s.libraryHandler.HandleLibrary)))

	// Protected author routes
	s.mux.Handle("/api/authors", s.requireAuth(http.HandlerFunc(s.authorHandler.HandleAuthors)))
	s.mux.Handle("/api/authors/", s.requireAuth(http.HandlerFunc(s.authorHandler.HandleAuthor)))

	// Protected series routes
	s.mux.Handle("/api/series", s.requireAuth(http.HandlerFunc(s.seriesHandler.HandleSeriesList)))
	s.mux.Handle("/api/series/", s.requireAuth(http.HandlerFunc(s.seriesHandler.HandleSeries)))

	// Protected book routes
	s.mux.Handle("/api/books", s.requireAuth(http.HandlerFunc(s.bookHandler.HandleBooks)))
	s.mux.Handle("/api/books/", s.requireAuth(http.HandlerFunc(s.bookHandler.HandleBookRoutes)))

	// Protected book file routes
	s.mux.Handle("/api/book-files/", s.requireAuth(http.HandlerFunc(s.bookFileHandler.HandleBookFile)))

	// Health check
	s.mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"ok"}`)
	})
	s.setupFrontend()
}

// oidcRoute returns a handler that forwards to the OIDC handler method if OIDC is configured,
// or responds with a 404 JSON error if it is not.
func (s *Server) oidcRoute(fn func(*handlers.OIDCHandler, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler := s.oidcHandler
		if handler == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"error":"OIDC not configured"}`)
			return
		}
		fn(handler, w, r)
	}
}

func (s *Server) setupFrontend() {
	// Serve the embedded frontend SPA
	frontendFS, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		slog.Error("failed to setup frontend filesystem", slog.Any("error", err))
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
