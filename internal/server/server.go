// Package server wires together the HTTP server, all handler structs,
// middleware chains (auth, rate limiting, request IDs, logging, tracing),
// and serves the embedded Svelte frontend as a single self-contained binary.
package server

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/authstore"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/handlers"
	"github.com/amalgamated-tools/biblioteka/internal/handlers/middleware"
	"github.com/amalgamated-tools/biblioteka/internal/llm"
	"github.com/amalgamated-tools/biblioteka/internal/llm/registry"
	"github.com/amalgamated-tools/biblioteka/internal/otel"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/pubsub"
	"github.com/amalgamated-tools/biblioteka/internal/worker"

	_ "github.com/amalgamated-tools/biblioteka/docs/swagger"

	goauthhandler "github.com/amalgamated-tools/goauth/handler"

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
	authCfg := auth.Config{CookieName: auth.TokenCookieName(), APIKeyPrefix: auth.APIKeyPrefix}
	jwtOnlyCfg := auth.Config{CookieName: auth.TokenCookieName()}

	if s.requireAuth == nil {
		s.requireAuth = auth.Middleware(s.JWT, authCfg, apiKeyAdapter)
	}

	if s.requireJWTAuth == nil {
		s.requireJWTAuth = auth.Middleware(s.JWT, jwtOnlyCfg, nil)
	}

	if s.requireAdmin == nil {
		s.requireAdmin = auth.AdminMiddleware(s.JWT, userAdapter, authCfg, apiKeyAdapter)
	}

	if s.authLimiter == nil {
		var trustedProxies []*net.IPNet
		if raw := os.Getenv("TRUSTED_PROXIES"); raw != "" {
			var err error
			trustedProxies, err = auth.ParseTrustedProxyCIDRs(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid TRUSTED_PROXIES: %w", err)
			}
			slog.InfoContext(ctx, "rate limiter trusting proxies from X-Forwarded-For", slog.Int(otelkeys.Count, len(trustedProxies)))
		}
		if len(trustedProxies) > 0 {
			s.authLimiter = auth.NewRateLimiterWithTrustedProxies(5, 10, trustedProxies)
		} else {
			s.authLimiter = auth.NewRateLimiter(5, 10)
		}
	}

	// Determine cookie security mode: secure by default, can be disabled for local dev
	secureCookies := os.Getenv("SECURE_COOKIES") != "false"
	s.secureCookies = secureCookies

	// Disable signup if DISABLE_SIGNUP=true; signup is enabled by default.
	disableSignup := os.Getenv("DISABLE_SIGNUP") == "true"

	s.authHandler = &handlers.AuthHandler{
		AuthHandler: goauthhandler.AuthHandler{
			Users:         userAdapter,
			JWT:           s.JWT,
			CookieName:    auth.TokenCookieName(),
			SecureCookies: secureCookies,
			DisableSignup: disableSignup,
		},
		DB: s.DB,
	}

	// Initialize WebAuthn for passkey support. RPID and origins must match the
	// deployment domain; they default to localhost for local development.
	s.passkeyHandler = newPasskeyHandler(ctx, s.DB, s.JWT, secureCookies)
	s.adminHandler = &handlers.AdminHandler{DB: s.DB}
	s.libraryHandler = &handlers.LibraryHandler{DB: s.DB}
	if s.Worker != nil {
		s.libraryHandler.Enqueuer = s.Worker
	}
	s.authorHandler = &handlers.AuthorHandler{DB: s.DB}
	s.seriesHandler = &handlers.SeriesHandler{DB: s.DB}
	s.readingListHandler = &handlers.ReadingListHandler{DB: s.DB}
	s.bookHandler = &handlers.BookHandler{DB: s.DB}

	// Always wire MetadataHandler so GET/apply/reject endpoints work without
	// a background worker. Only Enqueuer and Subscriber are conditional.
	metadataHandler := &handlers.MetadataHandler{DB: s.DB}
	if s.Worker != nil {
		s.bookHandler.Enqueuer = s.Worker
		metadataHandler.Enqueuer = s.Worker

		// Create a pub/sub subscriber for SSE metadata events.
		redisURL := os.Getenv("REDIS_URL")
		if redisURL == "" {
			redisURL = "redis://localhost:6379"
		}
		psClient, err := pubsub.NewClient(redisURL)
		if err != nil {
			slog.WarnContext(ctx, "failed to create pubsub client for metadata events; SSE streaming disabled",
				slog.Any(otelkeys.Error, err),
			)
		} else {
			s.shutdownFuncs = append(s.shutdownFuncs, func(_ context.Context) error {
				return psClient.Close()
			})
			metadataHandler.Subscriber = psClient
		}
	}

	// Wire up the LLM provider if configured in settings.
	// NOTE: LLM config is read once at startup. Changes via /api/config/llm
	// require a server restart to take effect (communicated via restart_required
	// in the PUT response).
	llmResult := llm.Bootstrap(ctx, s.DB, llm.BootstrapSettings{
		Enabled:  db.SettingLLMEnabled,
		Provider: db.SettingLLMProvider,
		Endpoint: db.SettingLLMEndpoint,
		Model:    db.SettingLLMModel,
	}, registry.DefaultFactories())
	if llmResult.Provider != nil {
		metadataHandler.LLMProvider = llmResult.Provider
		slog.InfoContext(ctx, "LLM provider configured",
			slog.String(otelkeys.Source, llmResult.ProviderName),
		)
	}

	s.bookHandler.MetadataHandler = metadataHandler
	s.tagHandler = &handlers.TagHandler{DB: s.DB}
	s.bookFileHandler = &handlers.BookFileHandler{DB: s.DB, Secrets: secretEncrypter}
	s.auditLogHandler = &handlers.AuditLogHandler{DB: s.DB}
	s.opdsHandler = &handlers.OPDSHandler{DB: s.DB}
	s.opdsCredentialHandler = &handlers.OPDSCredentialHandler{DB: s.DB}
	s.kosyncHandler = &handlers.KOSyncHandler{DB: s.DB}
	s.readingProgressHandler = &handlers.ReadingProgressHandler{DB: s.DB}
	s.calibreImportHandler = &handlers.CalibreImportHandler{DB: s.DB}
	s.apiKeyHandler = &handlers.APIKeyHandler{
		APIKeyHandler: goauthhandler.APIKeyHandler{
			APIKeys: apiKeyAdapter,
			Prefix:  auth.APIKeyPrefix,
			URLParamFunc: func(r *http.Request, key string) string {
				rest := strings.TrimPrefix(r.URL.Path, "/api/api-keys/")
				rest = strings.TrimSuffix(rest, "/")
				if strings.Contains(rest, "/") {
					return ""
				}
				return rest
			},
		},
		DB: s.DB,
	}
	s.koboHandler = &handlers.KoboHandler{DB: s.DB}
	s.koboHandler.RegisterRoutes()
	s.groupHandler = &handlers.GroupHandler{DB: s.DB}
	s.statsHandler = &handlers.StatsHandler{DB: s.DB}
	s.recommendationHandler = &handlers.RecommendationHandler{DB: s.DB}
	s.requireKoboAuth = auth.KoboTokenAuthMiddleware(&koboDBAdapter{db: s.DB})
	protocolCredAdapter := &protocolCredDBAdapter{db: s.DB}
	s.requireOPDSAuth = auth.OPDSBasicAuthMiddleware(protocolCredAdapter)
	s.requireKOSyncAuth = auth.KOSyncHeaderAuthMiddleware(protocolCredAdapter)
	s.configHandler = &handlers.ConfigHandler{
		DB:               s.DB,
		Secrets:          secretEncrypter,
		IsOIDCConfigured: func() bool { return s.oidcHandler != nil },
		OnOIDCConfigSet: func(ctx context.Context, issuerURL, clientID, clientSecret, redirectURI string) error {
			oidcHandler, err := handlers.NewOIDCHandler(ctx, userAdapter, s.JWT, issuerURL, clientID, clientSecret, redirectURI, auth.TokenCookieName(), secureCookies)
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
			return nil, errors.New("OIDC_ISSUER_URL is set but one or more of OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, or OIDC_REDIRECT_URI is missing")
		}
		oidcHandler, err := handlers.NewOIDCHandler(ctx, userAdapter, s.JWT, issuer, clientID, clientSecret, redirectURI, auth.TokenCookieName(), secureCookies)
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
			oidcHandler, err := handlers.NewOIDCHandler(ctx, userAdapter, s.JWT, dbIssuer, dbClientID, dbClientSecret, dbRedirectURI, auth.TokenCookieName(), secureCookies)
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

// protocolCredDBAdapter bridges *db.DB to the auth.OPDSCredentialChecker
// and auth.KOSyncCredentialChecker interfaces.
type protocolCredDBAdapter struct {
	db *db.DB
}

// GetOPDSCredential looks up the OPDS credential for the given username and
// returns the associated user ID and bcrypt-hashed password for the auth
// middleware to verify. Returns sql.ErrNoRows (wrapped) when not found.
func (a *protocolCredDBAdapter) GetOPDSCredential(ctx context.Context, username string) (*auth.ProtocolCredentialResult, error) {
	cred, err := a.db.GetOPDSCredentialByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(
				ctx,
				"OPDS credential not found",
				slog.String(otelkeys.OPDSUsername, username),
			)
		} else {
			slog.ErrorContext(
				ctx,
				"failed to get OPDS credential",
				slog.String(otelkeys.OPDSUsername, username),
				slog.Any(otelkeys.Error, err),
			)
		}
		return nil, fmt.Errorf("failed to get OPDS credential for username %s: %w", username, err)
	}
	return &auth.ProtocolCredentialResult{
		UserID:       cred.UserID,
		PasswordHash: cred.PasswordHash,
	}, nil
}

// GetKOSyncCredential looks up the KOSync credential for the given username
// and returns the associated user ID and bcrypt-hashed password for the auth
// middleware to verify. Returns sql.ErrNoRows (wrapped) when not found.
func (a *protocolCredDBAdapter) GetKOSyncCredential(ctx context.Context, username string) (*auth.ProtocolCredentialResult, error) {
	cred, err := a.db.GetKOSyncCredentialByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(
				ctx,
				"KOSync credential not found",
				slog.String(otelkeys.KOSyncUsername, username),
			)
		} else {
			slog.ErrorContext(
				ctx,
				"failed to get KOSync credential",
				slog.String(otelkeys.KOSyncUsername, username),
				slog.Any(otelkeys.Error, err),
			)
		}
		return nil, fmt.Errorf("failed to get KOSync credential for username %s: %w", username, err)
	}
	return &auth.ProtocolCredentialResult{
		UserID:       cred.UserID,
		PasswordHash: cred.PasswordHash,
	}, nil
}

// koboDBAdapter bridges *db.DB to the auth.KoboTokenChecker interface.
type koboDBAdapter struct {
	db *db.DB
}

// GetKoboTokenByToken hashes the raw Kobo token and looks up the matching
// record, returning the associated user ID for injection into the request
// context by KoboAuthMiddleware. Returns sql.ErrNoRows (wrapped) when the
// token is not found.
func (a *koboDBAdapter) GetKoboTokenByToken(ctx context.Context, token string) (*auth.KoboTokenResult, error) {
	tokenHash := auth.HashKoboToken(token)
	t, err := a.db.GetKoboTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(
				ctx,
				"Kobo token not found",
				slog.String(otelkeys.TokenHash, tokenHash),
			)
		} else {
			slog.ErrorContext(
				ctx,
				"failed to get Kobo token",
				slog.String(otelkeys.TokenHash, tokenHash),
				slog.Any(otelkeys.Error, err),
			)
		}
		return nil, fmt.Errorf("failed to get Kobo token for token hash %s: %w", tokenHash, err)
	}
	return &auth.KoboTokenResult{
		UserID: t.UserID,
	}, nil
}
