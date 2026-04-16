package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/authstore"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/go-webauthn/webauthn/webauthn"
	goauthhandler "github.com/amalgamated-tools/goauth/handler"
)

// newPasskeyHandler creates a PasskeyHandler configured from environment variables.
func newPasskeyHandler(ctx context.Context, database *db.DB, jwt *auth.JWTManager, secureCookies bool) *goauthhandler.PasskeyHandler {
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

	userAdapter := &authstore.UserAdapter{DB: database}
	passkeyAdapter := &authstore.PasskeyAdapter{DB: database}

	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: rpName,
		RPID:          rpID,
		RPOrigins:     rpOrigins,
	})
	if err != nil {
		slog.WarnContext(ctx, "failed to initialize WebAuthn; passkeys disabled",
			slog.Any(otelkeys.Error, err),
		)
		return &goauthhandler.PasskeyHandler{
			Users: userAdapter, Passkeys: passkeyAdapter,
			WebAuthn: nil, JWT: jwt,
			CookieName: auth.TokenCookieName(), SecureCookies: secureCookies,
			URLParamFunc: func(r *http.Request, key string) string { return "" },
		}
	}

	slog.InfoContext(ctx, "WebAuthn passkeys enabled",
		slog.String(otelkeys.WebAuthnRPID, rpID),
		slog.String(otelkeys.WebAuthnRPName, rpName),
	)

	return &goauthhandler.PasskeyHandler{
		Users: userAdapter, Passkeys: passkeyAdapter,
		WebAuthn: wa, JWT: jwt,
		CookieName: auth.TokenCookieName(), SecureCookies: secureCookies,
		URLParamFunc: func(r *http.Request, key string) string {
			// biblioteka uses stdlib mux, not Chi — extract from URL path manually
			rest := strings.TrimPrefix(r.URL.Path, "/api/auth/passkey/credentials/")
			rest = strings.TrimSuffix(rest, "/")
			if strings.Contains(rest, "/") {
				return ""
			}
			return rest
		},
	}
}
