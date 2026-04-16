// Package auth provides protocol-specific authentication middleware (OPDS,
// KOSync, Kobo) and re-exports common auth symbols from goauth for backward
// compatibility with the rest of the biblioteka codebase.
//
// Core auth logic (JWT, middleware, rate limiting, crypto) lives in
// github.com/amalgamated-tools/goauth/auth.
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"

	goauth "github.com/amalgamated-tools/goauth/auth"
	"golang.org/x/crypto/bcrypt"
)

// Re-export types from goauth.
type (
	JWTManager      = goauth.JWTManager
	Claims          = goauth.Claims
	RateLimiter     = goauth.RateLimiter
	SecretEncrypter = goauth.SecretEncrypter
	AdminChecker    = goauth.AdminChecker
	Config          = goauth.Config
	APIKeyValidator = goauth.APIKeyStore
	APIKey          = goauth.APIKey
)

// Re-export functions.
var (
	NewJWTManager                    = goauth.NewJWTManager
	Middleware                       = goauth.Middleware
	AdminMiddleware                  = goauth.AdminMiddleware
	NewRateLimiter                   = goauth.NewRateLimiter
	NewRateLimiterWithTrustedProxies = goauth.NewRateLimiterWithTrustedProxies
	ParseTrustedProxyCIDRs           = goauth.ParseTrustedProxyCIDRs
	HashAPIKey                       = goauth.HashHighEntropyToken
	GenerateRandomHex                = goauth.GenerateRandomHex
)

// Re-export constants.
const (
	MinSecretLength = goauth.MinSecretLength
	APIKeyPrefix    = "bib_"
)

// Re-export error sentinels.
var (
	ErrInvalidToken = goauth.ErrInvalidToken
	ErrExpiredToken = goauth.ErrExpiredToken
	ErrEmailExists  = goauth.ErrEmailExists
)

// TokenCookieName returns the cookie name.
func TokenCookieName() string { return "biblioteka_token" }

// contextKey is the type used for context keys in protocol middleware.
type contextKey string

// jsonError writes a JSON error response. Used by protocol middleware.
func jsonError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// mustGenerateDummyBcryptHash generates a bcrypt hash for timing-safe comparisons.
func mustGenerateDummyBcryptHash(secret string, name string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), goauth.BcryptCost)
	if err != nil {
		panic(fmt.Errorf("generate dummy %s bcrypt hash: %w", name, err))
	}
	return hash
}
