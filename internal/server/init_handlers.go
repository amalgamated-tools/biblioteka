package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/authstore"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/handlers"
	"github.com/amalgamated-tools/biblioteka/internal/llm"
	"github.com/amalgamated-tools/biblioteka/internal/llm/registry"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/pubsub"

	goauthhandler "github.com/amalgamated-tools/goauth/handler"
)

func (s *Server) initHandlers(
	ctx context.Context,
	secretEncrypter *auth.SecretEncrypter,
	apiKeyAdapter *authstore.APIKeyAdapter,
	userAdapter *authstore.UserAdapter,
) error {
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
				return fmt.Errorf("invalid TRUSTED_PROXIES: %w", err)
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
	s.secureCookies = os.Getenv("SECURE_COOKIES") != "false"

	// Disable signup if DISABLE_SIGNUP=true; signup is enabled by default.
	disableSignup := os.Getenv("DISABLE_SIGNUP") == "true"

	s.authHandler = &handlers.AuthHandler{
		AuthHandler: goauthhandler.AuthHandler{
			Users:         userAdapter,
			JWT:           s.JWT,
			CookieName:    auth.TokenCookieName(),
			SecureCookies: s.secureCookies,
			DisableSignup: disableSignup,
		},
		DB: s.DB,
	}

	// Initialize WebAuthn for passkey support. RPID and origins must match the
	// deployment domain; they default to localhost for local development.
	s.passkeyHandler = newPasskeyHandler(ctx, s.DB, s.JWT, s.secureCookies)
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
			URLParamFunc: func(r *http.Request, _ string) string {
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
			oidcHandler, err := handlers.NewOIDCHandler(ctx, userAdapter, s.JWT, issuerURL, clientID, clientSecret, redirectURI, auth.TokenCookieName(), s.secureCookies)
			if err != nil {
				slog.ErrorContext(ctx, "failed to initialize OIDC provider with new settings", slog.Any(otelkeys.Error, err))
				return fmt.Errorf("failed to initialize OIDC provider with new settings: %w", err)
			}
			s.oidcHandler = oidcHandler
			return nil
		},
	}

	return nil
}
