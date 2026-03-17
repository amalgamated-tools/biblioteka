package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/handlers"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"

	"github.com/hibiken/asynqmon"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (s *Server) setupRoutes(ctx context.Context) {
	// Public auth routes (rate-limited)
	s.mux.HandleFunc("/api/auth/signup", s.authLimiter.Limit(s.authHandler.Signup))
	s.mux.HandleFunc("/api/auth/login", s.authLimiter.Limit(s.authHandler.Login))
	s.mux.HandleFunc("/api/auth/logout", s.authLimiter.Limit(s.authHandler.Logout))

	// OIDC auth routes — always registered, check handler at request time
	s.mux.HandleFunc("/api/auth/oidc/enabled", s.handleOIDCEnabled)
	s.mux.HandleFunc("/api/auth/oidc/login", s.authLimiter.Limit(s.oidcRoute((*handlers.OIDCHandler).Login)))
	s.mux.HandleFunc("/api/auth/oidc/callback", s.authLimiter.Limit(s.oidcRoute((*handlers.OIDCHandler).Callback)))
	s.mux.HandleFunc("/api/auth/oidc/link", s.authLimiter.Limit(s.oidcRoute((*handlers.OIDCHandler).Link)))
	s.mux.Handle("/api/auth/oidc/link-nonce", s.requireAuth(http.HandlerFunc(s.oidcRoute((*handlers.OIDCHandler).CreateLinkNonce))))

	// Protected auth routes
	s.mux.Handle("/api/auth/me", s.requireAuth(http.HandlerFunc(s.authHandler.Me)))
	s.mux.Handle("/api/auth/password", s.requireJWTAuth(http.HandlerFunc(s.authHandler.ChangePassword)))

	// Protected config routes (JWT-only: sensitive server configuration)
	s.mux.Handle("/api/config/status", s.requireJWTAuth(http.HandlerFunc(s.configHandler.HandleConfigStatus)))
	s.mux.Handle("/api/config/oidc", s.requireJWTAuth(http.HandlerFunc(s.configHandler.HandleOIDCConfig)))
	s.mux.Handle("/api/config/smtp", s.requireJWTAuth(http.HandlerFunc(s.configHandler.HandleSMTPConfig)))
	s.mux.Handle("/api/config/smtp/test", s.requireJWTAuth(s.authLimiter.Limit(s.configHandler.HandleSMTPTest)))

	// Protected admin routes (JWT-only: user management)
	s.mux.Handle("/api/admin/users", s.requireJWTAuth(http.HandlerFunc(s.adminHandler.HandleListUsers)))
	s.mux.Handle("/api/admin/users/", s.requireJWTAuth(http.HandlerFunc(s.adminHandler.HandleSetAdmin)))

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

	// Protected audit log routes (admin only)
	s.mux.Handle("/api/audit-logs", s.requireAuth(http.HandlerFunc(s.auditLogHandler.HandleAuditLogs)))

	// OPDS credential management (JWT-only: credential management)
	s.mux.Handle("/api/opds/credentials", s.requireJWTAuth(http.HandlerFunc(s.opdsCredentialHandler.HandleOPDSCredentials)))

	// OPDS feed routes (Basic Auth)
	s.mux.Handle("/opds", s.requireOPDSAuth(http.HandlerFunc(s.opdsHandler.HandleOPDS)))
	s.mux.Handle("/opds/", s.requireOPDSAuth(http.HandlerFunc(s.opdsHandler.HandleOPDS)))

	// KOSync credential management (JWT-only: credential management)
	s.mux.Handle("/api/kosync/credentials", s.requireJWTAuth(http.HandlerFunc(s.kosyncHandler.HandleKOSyncCredentials)))

	// KOReader kosync-compatible progress sync endpoints.
	// POST /api/user/create — KOReader always tries to register; we return 409 so
	// it falls through to /api/user/auth.  Users set up credentials via the web UI.
	s.mux.HandleFunc("/api/user/create", s.kosyncHandler.HandleKOSyncUserCreate)
	// GET /api/user/auth — verified by the KOSync header auth middleware.
	s.mux.HandleFunc("/api/user/auth", s.authLimiter.Limit(s.requireKOSyncAuth(http.HandlerFunc(s.kosyncHandler.HandleKOSyncUserAuth)).ServeHTTP))
	// PUT /api/syncs/progress and GET /api/syncs/progress/{document}.
	s.mux.HandleFunc("/api/syncs/progress", s.authLimiter.Limit(s.requireKOSyncAuth(http.HandlerFunc(s.kosyncHandler.HandleKOSyncProgress)).ServeHTTP))
	s.mux.HandleFunc("/api/syncs/progress/", s.authLimiter.Limit(s.requireKOSyncAuth(http.HandlerFunc(s.kosyncHandler.HandleKOSyncProgress)).ServeHTTP))

	// Protected API key routes (JWT-only: API keys cannot manage other API keys)
	s.mux.Handle("/api/api-keys", s.requireJWTAuth(http.HandlerFunc(s.apiKeyHandler.HandleAPIKeys)))
	s.mux.Handle("/api/api-keys/", s.requireJWTAuth(http.HandlerFunc(s.apiKeyHandler.HandleAPIKey)))

	// Kobo sync token management (JWT-only: same constraint as API keys)
	s.mux.Handle("/api/kobo/tokens", s.requireJWTAuth(http.HandlerFunc(s.koboHandler.HandleKoboTokens)))
	s.mux.Handle("/api/kobo/tokens/", s.requireJWTAuth(http.HandlerFunc(s.koboHandler.HandleKoboToken)))

	// Kobo device API — token auth via middleware, sub-mux handles routing
	s.mux.Handle("/kobo/", s.requireKoboAuth(s.koboHandler))

	// Health check
	s.mux.HandleFunc("/api/health", s.handleHealth)

	// Version
	s.mux.HandleFunc("/api/version", s.handleVersion)

	// Swagger UI (public, with restrictive security headers)
	swaggerHandler := swaggerSecurityHeaders(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	s.mux.Handle("/swagger", http.RedirectHandler("/swagger/", http.StatusMovedPermanently))
	s.mux.Handle("/swagger/", swaggerHandler)

	// Asynq monitoring dashboard (admin only, supports cookie auth for browser access)
	if s.Worker != nil {
		mon := asynqmon.New(asynqmon.Options{
			RootPath:     "/asynqmon",
			RedisConnOpt: s.Worker.RedisConnOpt(),
		})
		s.mux.Handle(mon.RootPath()+"/", s.requireAdmin(mon))
	}

	s.setupFrontend(ctx)
}

// swaggerSecurityHeaders wraps a handler with restrictive CORS and CSP headers
// to limit cross-origin access to the publicly-available Swagger documentation.
func swaggerSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")

		// For requests with an Origin header, do not enable CORS; browsers will block
		// cross-origin access because no Access-Control-Allow-Origin header is sent.
		if origin := r.Header.Get("Origin"); origin != "" {
			// Ensure we do not overwrite any existing Vary dimensions; merge Origin in.
			existingVary := w.Header().Get("Vary")
			if existingVary == "" {
				w.Header().Set("Vary", "Origin")
			} else if !strings.Contains(existingVary, "Origin") {
				w.Header().Set("Vary", existingVary+", Origin")
			}
			// No Access-Control-Allow-Origin header → browser blocks the response.
		}

		next.ServeHTTP(w, r)
	})
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

// handleOIDCEnabled godoc
//
//	@Summary		Check if OIDC is enabled
//	@Description	Returns whether OIDC authentication is configured on this server
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	oidcEnabledResponse
//	@Router			/auth/oidc/enabled [get]
func (s *Server) handleOIDCEnabled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"}); err != nil {
			slog.ErrorContext(r.Context(), "failed to encode OIDC enabled method not allowed response", slog.Any(otelkeys.Error, err))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := oidcEnabledResponse{
		Enabled: s.oidcHandler != nil,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode OIDC enabled response", slog.Any(otelkeys.Error, err))
	}
}

type healthResponse struct {
	Status string `json:"status"`
}

type versionResponse struct {
	Version string `json:"version"`
}

type oidcEnabledResponse struct {
	Enabled bool `json:"enabled"`
}

// handleHealth godoc
//
//	@Summary		Health check
//	@Description	Returns server health status
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	healthResponse
//	@Router			/health [get]
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		w.WriteHeader(http.StatusMethodNotAllowed)

		resp := map[string]string{
			"error": "method not allowed",
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.ErrorContext(r.Context(), "failed to encode health method not allowed response", slog.Any(otelkeys.Error, err))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := healthResponse{
		Status: "ok",
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode health response", slog.Any(otelkeys.Error, err))
	}
}

// checkSystemEndpointMethod validates that the request method is one of the allowed methods.
// If not, it writes a JSON 405 Method Not Allowed response and returns false.
// Callers should return immediately when this function returns false.
func checkSystemEndpointMethod(w http.ResponseWriter, r *http.Request, logMessage string, allowedMethods ...string) bool {
	for _, m := range allowedMethods {
		if r.Method == m {
			return true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	w.WriteHeader(http.StatusMethodNotAllowed)

	resp := map[string]string{
		"error": "method not allowed",
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), logMessage, slog.Any(otelkeys.Error, err))
	}

	return false
}

// handleVersion godoc
//
//	@Summary		Get server version
//	@Description	Returns the server version
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	versionResponse
//	@Router			/version [get]
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if !checkSystemEndpointMethod(w, r, "failed to encode version method not allowed response", http.MethodGet, http.MethodHead) {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := versionResponse{
		Version: s.version,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode version response", slog.Any(otelkeys.Error, err))
	}
}
