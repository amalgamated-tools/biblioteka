package server

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/handlers"
	"github.com/amalgamated-tools/biblioteka/internal/handlers/middleware"
	"github.com/amalgamated-tools/biblioteka/internal/otel"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/worker"

	_ "github.com/amalgamated-tools/biblioteka/docs"

	"github.com/justinas/alice"
	"golang.org/x/sync/errgroup"
)

//go:embed all:dist
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
	version string

	DB  *db.DB
	JWT *auth.JWTManager

	Worker *worker.Worker

	oidcHandler           *handlers.OIDCHandler
	authHandler           *handlers.AuthHandler
	configHandler         *handlers.ConfigHandler
	adminHandler          *handlers.AdminHandler
	libraryHandler        *handlers.LibraryHandler
	authorHandler         *handlers.AuthorHandler
	seriesHandler         *handlers.SeriesHandler
	bookHandler           *handlers.BookHandler
	bookFileHandler       *handlers.BookFileHandler
	auditLogHandler       *handlers.AuditLogHandler
	apiKeyHandler         *handlers.APIKeyHandler
	opdsHandler           *handlers.OPDSHandler
	opdsCredentialHandler *handlers.OPDSCredentialHandler
	koboHandler           *handlers.KoboHandler
	kosyncHandler         *handlers.KOSyncHandler
	requireAuth           func(http.Handler) http.Handler
	requireJWTAuth        func(http.Handler) http.Handler
	requireAdmin          func(http.Handler) http.Handler
	requireOPDSAuth       func(http.Handler) http.Handler
	requireKoboAuth       func(http.Handler) http.Handler
	requireKOSyncAuth     func(http.Handler) http.Handler
	authLimiter           *auth.RateLimiter
	mux                   *http.ServeMux
	httpServer            *http.Server
	shutdownFuncs         []ShutdownFunc
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
		s.requireAuth = auth.Middleware(s.JWT, s.DB)
	}

	if s.requireJWTAuth == nil {
		s.requireJWTAuth = auth.Middleware(s.JWT, nil)
	}

	if s.requireAdmin == nil {
		var adminChecker auth.AdminChecker = s.DB
		var apiKeyValidator auth.APIKeyValidator = s.DB
		s.requireAdmin = auth.AdminMiddleware(s.JWT, adminChecker, apiKeyValidator)
	}

	if s.authLimiter == nil {
		s.authLimiter = auth.NewRateLimiter(5, 10)
	}

	// Determine cookie security mode: secure by default, can be disabled for local dev
	secureCookies := os.Getenv("SECURE_COOKIES") != "false"

	s.authHandler = &handlers.AuthHandler{DB: s.DB, JWT: s.JWT, SecureCookies: secureCookies}
	s.adminHandler = &handlers.AdminHandler{DB: s.DB}
	s.libraryHandler = &handlers.LibraryHandler{DB: s.DB}
	if s.Worker != nil {
		s.libraryHandler.Enqueuer = s.Worker
	}
	s.authorHandler = &handlers.AuthorHandler{DB: s.DB}
	s.seriesHandler = &handlers.SeriesHandler{DB: s.DB}
	s.bookHandler = &handlers.BookHandler{DB: s.DB}
	s.bookFileHandler = &handlers.BookFileHandler{DB: s.DB}
	s.auditLogHandler = &handlers.AuditLogHandler{DB: s.DB}
	s.opdsHandler = &handlers.OPDSHandler{DB: s.DB}
	s.opdsCredentialHandler = &handlers.OPDSCredentialHandler{DB: s.DB}
	s.kosyncHandler = &handlers.KOSyncHandler{DB: s.DB}
	s.apiKeyHandler = &handlers.APIKeyHandler{DB: s.DB}
	s.koboHandler = &handlers.KoboHandler{DB: s.DB}
	s.koboHandler.RegisterRoutes()
	s.requireKoboAuth = auth.KoboTokenAuthMiddleware(&koboDBAdapter{db: s.DB})
	s.requireOPDSAuth = auth.OPDSBasicAuthMiddleware(&opdsDBAdapter{db: s.DB})
	s.requireKOSyncAuth = auth.KOSyncHeaderAuthMiddleware(&kosyncDBAdapter{db: s.DB})
	s.configHandler = &handlers.ConfigHandler{
		DB:               s.DB,
		IsOIDCConfigured: func() bool { return s.oidcHandler != nil },
		OnOIDCConfigSet: func(ctx context.Context, issuerURL, clientID, clientSecret, redirectURI string) error {
			oidcHandler, err := handlers.NewOIDCHandler(ctx, s.DB, s.JWT, issuerURL, clientID, clientSecret, redirectURI, secureCookies)
			if err != nil {
				slog.ErrorContext(ctx, "failed to initialize OIDC provider with new settings", slog.Any(otelkeys.Error, err))
				return fmt.Errorf("failed to initialize OIDC provider with new settings: %w", err)
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
		slog.InfoContext(ctx, "OIDC authentication enabled", slog.String(otelkeys.Issuer, issuer))
	} else if dbIssuer, err := s.DB.GetSetting(ctx, "oidc_issuer_url"); err == nil && dbIssuer != "" {
		dbClientID, _ := s.DB.GetSetting(ctx, "oidc_client_id")
		dbClientSecret, _ := s.DB.GetSetting(ctx, "oidc_client_secret")
		dbRedirectURI, _ := s.DB.GetSetting(ctx, "oidc_redirect_uri")
		if dbClientID != "" && dbClientSecret != "" && dbRedirectURI != "" {
			oidcHandler, err := handlers.NewOIDCHandler(ctx, s.DB, s.JWT, dbIssuer, dbClientID, dbClientSecret, dbRedirectURI, secureCookies)
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

func (s *Server) Run(ctx context.Context) error {
	newctx, span := otel.StartTracer(ctx, "server.Run")
	defer span.End()
	slog.DebugContext(newctx, "Running server", slog.String(otelkeys.Address, s.Address))
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
			slog.ErrorContext(newctx, "HTTP server error", slog.Any(otelkeys.Error, err))
			s.shutdownFuncs = append(s.shutdownFuncs, func(ctx context.Context) error {
				slog.InfoContext(ctx, "Shutting down HTTP server due to error")
				return fmt.Errorf("HTTP server error: %w", err)
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

// opdsDBAdapter bridges *db.DB to the auth.OPDSCredentialChecker interface.
type opdsDBAdapter struct {
	db *db.DB
}

func (a *opdsDBAdapter) GetOPDSCredential(ctx context.Context, username string) (*auth.OPDSCredentialResult, error) {
	cred, err := a.db.GetOPDSCredentialByUsername(ctx, username)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get OPDS credential", slog.String(otelkeys.Username, username), slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("failed to get OPDS credential for username %s: %w", username, err)
	}
	return &auth.OPDSCredentialResult{
		UserID:       cred.UserID,
		PasswordHash: cred.PasswordHash,
	}, nil
}

// koboDBAdapter bridges *db.DB to the auth.KoboTokenChecker interface.
type koboDBAdapter struct {
	db *db.DB
}

func (a *koboDBAdapter) GetKoboTokenByToken(ctx context.Context, token string) (*auth.KoboTokenResult, error) {
	tokenHash := auth.HashKoboToken(token)
	t, err := a.db.GetKoboTokenByHash(ctx, tokenHash)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get Kobo token", slog.String(otelkeys.TokenHash, tokenHash), slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("failed to get Kobo token for token hash %s: %w", tokenHash, err)
	}
	return &auth.KoboTokenResult{
		UserID: t.UserID,
	}, nil
}

// kosyncDBAdapter bridges *db.DB to the auth.KOSyncCredentialChecker interface.
type kosyncDBAdapter struct {
	db *db.DB
}

func (a *kosyncDBAdapter) GetKOSyncCredential(ctx context.Context, username string) (*auth.KOSyncCredentialResult, error) {
	cred, err := a.db.GetKOSyncCredentialByUsername(ctx, username)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get KOSync credential", slog.String(otelkeys.Username, username), slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("failed to get KOSync credential for username %s: %w", username, err)
	}
	return &auth.KOSyncCredentialResult{
		UserID:       cred.UserID,
		PasswordHash: cred.PasswordHash,
	}, nil
}
