package server

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/handlers"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/go-webauthn/webauthn/webauthn"
)

// newPasskeyHandler creates a PasskeyHandler configured from environment variables.
//
// Required env vars for production deployments (defaults to localhost for dev):
//   - WEBAUTHN_RP_ID        — relying party ID (e.g. "mybooks.example.com")
//   - WEBAUTHN_RP_ORIGINS   — comma-separated allowed origins (e.g. "https://mybooks.example.com")
//   - WEBAUTHN_RP_NAME      — display name shown in the passkey dialog (default: "Biblioteka")
func newPasskeyHandler(ctx context.Context, database *db.DB, jwt *auth.JWTManager, secureCookies bool) *handlers.PasskeyHandler {
	rpID := os.Getenv("WEBAUTHN_RP_ID")
	if rpID == "" {
		rpID = "localhost"
	}

	rpOriginsRaw := os.Getenv("WEBAUTHN_RP_ORIGINS")
	var rpOrigins []string
	if rpOriginsRaw == "" {
		rpOrigins = []string{"http://localhost:8080"}
	} else {
		for _, o := range strings.Split(rpOriginsRaw, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				rpOrigins = append(rpOrigins, o)
			}
		}
	}

	rpName := os.Getenv("WEBAUTHN_RP_NAME")
	if rpName == "" {
		rpName = "Biblioteka"
	}

	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: rpName,
		RPID:          rpID,
		RPOrigins:     rpOrigins,
	})
	if err != nil {
		slog.WarnContext(ctx, "failed to initialize WebAuthn; passkeys disabled",
			slog.Any(otelkeys.Error, err),
		)
		return &handlers.PasskeyHandler{DB: database, WebAuthn: nil, JWT: jwt, SecureCookies: secureCookies}
	}

	slog.InfoContext(ctx, "WebAuthn passkeys enabled",
		slog.String(otelkeys.WebAuthnRPID, rpID),
		slog.String(otelkeys.WebAuthnRPName, rpName),
	)

	return &handlers.PasskeyHandler{DB: database, WebAuthn: wa, JWT: jwt, SecureCookies: secureCookies}
}
