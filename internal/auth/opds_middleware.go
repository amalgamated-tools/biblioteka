package auth

import (
	"bytes"
	"context"
	"encoding/xml"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// OPDSCredentialChecker is implemented by types that can look up OPDS credentials by username.
type OPDSCredentialChecker interface {
	GetOPDSCredential(ctx context.Context, username string) (*ProtocolCredentialResult, error)
}

// dummyOPDSBcryptHash is a precomputed valid bcrypt hash used for timing-safe
// comparisons when a username is not found, to mitigate username enumeration
// via timing attacks.
var dummyOPDSBcryptHash = mustGenerateDummyBcryptHash("dummy-opds-password", "OPDS")

// writeOPDSError writes an OPDS-compatible XML error response for authentication failures.
func writeOPDSError(_ context.Context, w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", `application/atom+xml;profile=opds-catalog;kind=navigation`)
	w.WriteHeader(status)

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	buf.WriteString(`<feed xmlns="http://www.w3.org/2005/Atom">`)
	buf.WriteString(`<title>Biblioteka OPDS Error</title>`)
	buf.WriteString(`<id>urn:biblioteka:opds:error</id>`)
	buf.WriteString(`<entry>`)
	buf.WriteString(`<title>Authentication Error</title>`)
	buf.WriteString(`<content type="text">`)
	_ = xml.EscapeText(&buf, []byte(message))
	buf.WriteString(`</content>`)
	buf.WriteString(`</entry>`)
	buf.WriteString(`</feed>`)

	_, _ = w.Write(buf.Bytes())
}

// OPDSBasicAuthMiddleware returns an HTTP middleware that validates OPDS
// credentials using HTTP Basic Authentication and injects the user ID into
// the request context.
func OPDSBasicAuthMiddleware(checker OPDSCredentialChecker) func(http.Handler) http.Handler {
	return bcryptCredMiddleware(bcryptCredConfig{
		protocolName: "OPDS",
		dummyHash:    dummyOPDSBcryptHash,
		usernameAttr: func(v string) slog.Attr { return slog.String(otelkeys.OPDSUsername, v) },
		extractCreds: func(r *http.Request) (username, secret string, ok bool) {
			username, secret, ok = r.BasicAuth()
			return username, secret, ok && username != ""
		},
		lookupCredential: lookupByUsername(checker.GetOPDSCredential, func(c *ProtocolCredentialResult) (string, string) {
			return c.UserID, c.PasswordHash
		}),
		writeMissing: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Biblioteka OPDS"`)
			writeOPDSError(r.Context(), w, http.StatusUnauthorized, "authentication required")
		},
		writeUnauthorized: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Biblioteka OPDS"`)
			writeOPDSError(r.Context(), w, http.StatusUnauthorized, "invalid credentials")
		},
		// writeServiceUnavailable is nil: OPDS treats all lookup errors as
		// "unknown user" and returns 401 to avoid leaking DB availability.
	})
}
