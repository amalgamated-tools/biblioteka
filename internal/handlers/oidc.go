package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	oidcStateCookieName    = "oidc_state"
	oidcVerifierCookieName = "oidc_verifier"
	oidcStateCookieTTL     = 5 * time.Minute
)

// linkNonce is a short-lived, single-use token mapping to a user ID.
type linkNonce struct {
	UserID    string
	ExpiresAt time.Time
}

// OIDCHandler holds dependencies for OIDC auth endpoints.
type OIDCHandler struct {
	DB            *db.DB
	JWT           *auth.JWTManager
	Provider      *oidc.Provider
	Config        oauth2.Config
	SecureCookies bool

	linkNonces   map[string]linkNonce
	linkNoncesMu sync.Mutex
}

// NewOIDCHandler creates a new OIDCHandler by performing OIDC discovery on the issuer URL.
func NewOIDCHandler(ctx context.Context, database *db.DB, jwt *auth.JWTManager, issuerURL, clientID, clientSecret, redirectURI string, secureCookies bool) (*OIDCHandler, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}

	return &OIDCHandler{
		DB:            database,
		JWT:           jwt,
		Provider:      provider,
		Config:        config,
		SecureCookies: secureCookies,
		linkNonces:    make(map[string]linkNonce),
	}, nil
}

// Login godoc
//
// @Summary		OIDC login
// @Description	Redirects to the OIDC provider's authorization endpoint
// @Tags			OIDC
// @Success		302	"Redirect to OIDC provider"
// @Failure		404	{object}	errorResponse	"OIDC not configured"
// @Failure		500	{object}	errorResponse
// @Router			/auth/oidc/login [get]
func (h *OIDCHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	slog.DebugContext(r.Context(), "initiating OIDC login flow")

	state, err := generateState()
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to generate OIDC state", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to initiate login")
		return
	}

	verifier := oauth2.GenerateVerifier()

	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   int(oidcStateCookieTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.SecureCookies,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     oidcVerifierCookieName,
		Value:    verifier,
		Path:     "/",
		MaxAge:   int(oidcStateCookieTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.SecureCookies,
	})

	http.Redirect(w, r, h.Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
