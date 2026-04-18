// Package server wires together the HTTP server, all handler structs,
// middleware chains (auth, rate limiting, request IDs, logging, tracing),
// and serves the embedded Svelte frontend as a single self-contained binary.
package server

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/authstore"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/handlers"
	"github.com/amalgamated-tools/biblioteka/internal/handlers/middleware"
	"github.com/amalgamated-tools/biblioteka/internal/otel"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/worker"

	_ "github.com/amalgamated-tools/biblioteka/docs/swagger"

	"github.com/justinas/alice"
	"golang.org/x/sync/errgroup"
)

//go:embed all:dist
var embeddedFiles embed.FS

const (
	// UserAgentHeader is the header name for the user agent.
	UserAgentHeader = "User-Agent"
	// HTTPWriteTimeout is the maximum duration before timing out writes of the response.
	// Set high enough to accommodate large uploads (e.g. 100 MB Calibre database imports).
	HTTPWriteTimeout = 120 * time.Second
	// HTTPReadTimeout is the maximum duration for reading the entire request, including the body.
	// Set high enough to accommodate large uploads on slow connections.
	HTTPReadTimeout = 120 * time.Second
	// HTTPIdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled.
	HTTPIdleTimeout = 30 * time.Second
	// HTTPRequestTimeout is the maximum duration for handling a single HTTP request.
	HTTPRequestTimeout = 10 * time.Second
	// ShutdownGracePeriod is the time we allow for graceful shutdown of the http server.
	// Should be longer than HTTPWriteTimeout, but shorter than the k8s terminationGracePeriodSeconds.
	ShutdownGracePeriod = 150 * time.Second
)

// ShutdownFunc is a function that takes a context and returns an error
type ShutdownFunc func(context.Context) error

// Server represents the HTTP server with embedded frontend
type Server struct {
	Address string
	port    int
	version string

	DB  *db.DB
	JWT *auth.JWTManager

	Worker *worker.Worker

	oidcHandler            *handlers.OIDCHandler
	authHandler            *handlers.AuthHandler
	passkeyHandler         *handlers.PasskeyHandler
	configHandler          *handlers.ConfigHandler
	adminHandler           *handlers.AdminHandler
	libraryHandler         *handlers.LibraryHandler
	authorHandler          *handlers.AuthorHandler
	seriesHandler          *handlers.SeriesHandler
	bookHandler            *handlers.BookHandler
	annotationHandler      *handlers.BookAnnotationHandler
	readingListHandler     *handlers.ReadingListHandler
	bookFileHandler        *handlers.BookFileHandler
	auditLogHandler        *handlers.AuditLogHandler
	apiKeyHandler          *handlers.APIKeyHandler
	tagHandler             *handlers.TagHandler
	opdsHandler            *handlers.OPDSHandler
	opdsCredentialHandler  *handlers.OPDSCredentialHandler
	koboHandler            *handlers.KoboHandler
	kosyncHandler          *handlers.KOSyncHandler
	groupHandler           *handlers.GroupHandler
	statsHandler           *handlers.StatsHandler
	readingProgressHandler *handlers.ReadingProgressHandler
	calibreImportHandler   *handlers.CalibreImportHandler
	recommendationHandler  *handlers.RecommendationHandler
	requireAuth            func(http.Handler) http.Handler
	requireJWTAuth         func(http.Handler) http.Handler
	requireAdmin           func(http.Handler) http.Handler
	requireOPDSAuth        func(http.Handler) http.Handler
	requireKoboAuth        func(http.Handler) http.Handler
	requireKOSyncAuth      func(http.Handler) http.Handler
	authLimiter            *auth.RateLimiter
	secureCookies          bool
	mux                    *http.ServeMux
	httpServer             *http.Server
	shutdownFuncs          []ShutdownFunc
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
	if s.Address == "" {
		s.Address = net.JoinHostPort("0.0.0.0", strconv.Itoa(s.port))
	}

	if s.DB == nil {
		database, err := db.SetupDatabase(ctx)
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
		jwtManager, err := auth.NewJWTManager(jwtSecret, 24*time.Hour, "biblioteka")
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JWT manager: %w", err)
		}
		s.JWT = jwtManager

		if jwtSecret == "" {
			slog.WarnContext(ctx, "JWT_SECRET not set, using random secret; all existing JWT tokens and any at-rest encrypted settings (SMTP password, OIDC client secret) will become invalid after a server restart")
		} else if len(jwtSecret) < auth.MinSecretLength {
			slog.WarnContext(ctx, "JWT_SECRET is shorter than the recommended minimum of 32 characters; a short secret weakens HMAC-SHA256 signing",
				slog.Int(otelkeys.JWTSecretLength, len(jwtSecret)),
			)
		}
	}

	secretEncrypter, err := s.JWT.NewSecretEncrypter()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize settings encrypter: %w", err)
	}

	apiKeyAdapter := &authstore.APIKeyAdapter{DB: s.DB}
	userAdapter := &authstore.UserAdapter{DB: s.DB}
	if err := s.initHandlers(ctx, secretEncrypter, apiKeyAdapter, userAdapter); err != nil {
		return nil, err
	}

	// Initialize OIDC handler if configured: env var first, then DB setting
	if issuer := os.Getenv("OIDC_ISSUER_URL"); issuer != "" {
		clientID := os.Getenv("OIDC_CLIENT_ID")
		clientSecret := os.Getenv("OIDC_CLIENT_SECRET")
		redirectURI := os.Getenv("OIDC_REDIRECT_URI")
		if clientID == "" || clientSecret == "" || redirectURI == "" {
			return nil, errors.New("OIDC_ISSUER_URL is set but one or more of OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, or OIDC_REDIRECT_URI is missing")
		}
		oidcHandler, err := handlers.NewOIDCHandler(ctx, userAdapter, s.JWT, issuer, clientID, clientSecret, redirectURI, auth.TokenCookieName(), s.secureCookies)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
		}
		s.oidcHandler = oidcHandler
		slog.InfoContext(ctx, "OIDC authentication enabled", slog.String(otelkeys.Issuer, issuer))
	} else if dbIssuer, err := s.DB.GetSetting(ctx, "oidc_issuer_url"); err == nil && dbIssuer != "" {
		dbClientID, _ := s.DB.GetSetting(ctx, "oidc_client_id")
		dbClientSecret, _ := s.DB.GetSetting(ctx, "oidc_client_secret")
		dbRedirectURI, _ := s.DB.GetSetting(ctx, "oidc_redirect_uri")
		// Decrypt the client secret stored in the database.
		if decrypted, decErr := secretEncrypter.Decrypt(dbClientSecret); decErr == nil {
			dbClientSecret = decrypted
		} else {
			dbClientSecret = ""
			slog.WarnContext(ctx, "failed to decrypt OIDC client secret from saved settings; skipping OIDC initialization from saved settings", slog.Any(otelkeys.Error, decErr))
		}
		if dbClientID != "" && dbClientSecret != "" && dbRedirectURI != "" {
			oidcHandler, err := handlers.NewOIDCHandler(ctx, userAdapter, s.JWT, dbIssuer, dbClientID, dbClientSecret, dbRedirectURI, auth.TokenCookieName(), s.secureCookies)
			if err != nil {
				slog.WarnContext(ctx, "failed to initialize OIDC from saved settings", slog.Any(otelkeys.Error, err))
			} else {
				s.oidcHandler = oidcHandler
				slog.InfoContext(ctx, "OIDC authentication enabled from saved settings", slog.String(otelkeys.Issuer, dbIssuer))
			}
		}
	}

	s.setupRoutes(ctx)
	return s, nil
}

// Run starts the HTTP server, applies request-ID, tracing, and logging
// middleware, and blocks until ctx is cancelled or a fatal server error
// occurs. A graceful shutdown is attempted with a ShutdownGracePeriod timeout.
func (s *Server) Run(ctx context.Context) error {
	newctx, span := otel.StartTracer(ctx, "server.Run")
	defer span.End()
	slog.DebugContext(newctx, "Running server", slog.String(otelkeys.Address, s.Address))
	ctx, cancel := context.WithCancel(newctx)
	defer cancel()

	chain := alice.New(
		middleware.RequestIDHandler,
		otel.TraceMiddleware,
		middleware.LoggingMiddleware,
		middleware.NewSecurityHeadersMiddleware(middleware.SecurityHeadersConfig{SecureCookies: s.secureCookies}),
	).Then(s.mux)

	s.httpServer = &http.Server{
		Addr:         s.Address,
		Handler:      chain,
		WriteTimeout: HTTPWriteTimeout,
		ReadTimeout:  HTTPReadTimeout,
		IdleTimeout:  HTTPIdleTimeout,
	}

	errCh := make(chan error, 1)

	go func() {
		err := s.httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	s.shutdownFuncs = append(s.shutdownFuncs, s.httpServer.Shutdown)

	select {
	case <-ctx.Done():
		return s.shutdown(ctx)
	case err := <-errCh:
		slog.ErrorContext(newctx, "HTTP server error", slog.Any(otelkeys.Error, err))
		s.shutdownFuncs = append(s.shutdownFuncs, func(ctx context.Context) error {
			slog.InfoContext(ctx, "Shutting down HTTP server due to error")
			return fmt.Errorf("HTTP server error: %w", err)
		})
		cancel()
		<-ctx.Done()
		return s.shutdown(newctx)
	}
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
