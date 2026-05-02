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
	"github.com/amalgamated-tools/biblioteka/internal/handlers"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	goauthhandler "github.com/amalgamated-tools/goauth/handler"
	"github.com/go-webauthn/webauthn/webauthn"
)

// newPasskeyHandler creates a PasskeyHandler configured from environment variables.
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
		for o := range strings.SplitSeq(rpOriginsRaw, ",") {
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

	// biblioteka uses stdlib mux, not Chi — extract from URL path manually.
	// Shared across both success and failure branches so credential management
	// (list/delete) works even when WebAuthn initialization fails.
	urlParamFunc := func(r *http.Request, key string) string {
		rest := strings.TrimPrefix(r.URL.Path, "/api/auth/passkey/credentials/")
		rest = strings.TrimSuffix(rest, "/")
		if strings.Contains(rest, "/") {
			return ""
		}
		return rest
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
		return &handlers.PasskeyHandler{
			PasskeyHandler: goauthhandler.PasskeyHandler{
				Users: userAdapter, Passkeys: passkeyAdapter,
				WebAuthn: nil, JWT: jwt,
				CookieName: auth.TokenCookieName(), SecureCookies: secureCookies,
				URLParamFunc: urlParamFunc,
			},
			DB: database,
		}
	}

	slog.InfoContext(ctx, "WebAuthn passkeys enabled",
		slog.String(otelkeys.WebAuthnRPID, rpID),
		slog.String(otelkeys.WebAuthnRPName, rpName),
	)

	return &handlers.PasskeyHandler{
		PasskeyHandler: goauthhandler.PasskeyHandler{
			Users: userAdapter, Passkeys: passkeyAdapter,
			WebAuthn: wa, JWT: jwt,
			CookieName: auth.TokenCookieName(), SecureCookies: secureCookies,
			URLParamFunc: urlParamFunc,
		},
		DB: database,
	}
}
