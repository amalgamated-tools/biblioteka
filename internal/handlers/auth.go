package handlers

import (
	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// AuthHandler holds dependencies for auth endpoints.
type AuthHandler struct {
	DB            *db.DB
	JWT           *auth.JWTManager
	SecureCookies bool
}
