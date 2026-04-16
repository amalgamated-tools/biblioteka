package auth

import goauth "github.com/amalgamated-tools/goauth/auth"

// hashHighEntropyToken delegates to goauth's exported function.
// Kept as a package-level unexported function for backward compatibility
// with protocol middleware in this package.
func hashHighEntropyToken(token string) string {
	return goauth.HashHighEntropyToken(token)
}

// Re-export symbols from goauth that this package's protocol middleware uses.
var (
	UserIDFromContext = goauth.UserIDFromContext
	ContextWithUserID = goauth.ContextWithUserID
)

const BcryptCost = goauth.BcryptCost
