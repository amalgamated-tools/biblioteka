package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"

	"github.com/stretchr/testify/require"
)

// newTestOIDCHandlerWithTokenResponse creates a handler wired to a mock OIDC
// server whose token endpoint always returns the given rawResponse map.
// Use this helper to test paths that depend on the raw token endpoint response,
// such as a missing or malformed id_token field.
func newTestOIDCHandlerWithTokenResponse(t *testing.T, rawResponse map[string]any) *OIDCHandler {
	t.Helper()

	var serverURL string
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                serverURL,
			"authorization_endpoint":                serverURL + "/authorize",
			"token_endpoint":                        serverURL + "/token",
			"jwks_uri":                              serverURL + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}); err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(jose.JSONWebKeySet{}); err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(rawResponse); err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
		}
	})

	srv := httptest.NewServer(mux)
	// serverURL must be assigned before oidc.NewProvider (below) is called,
	// because NewProvider fetches /.well-known/openid-configuration whose
	// handler closure captures serverURL by reference.
	serverURL = srv.URL
	t.Cleanup(srv.Close)

	provider, err := oidc.NewProvider(t.Context(), srv.URL)
	require.NoError(t, err, "newTestOIDCHandlerWithTokenResponse: oidc.NewProvider")

	return &OIDCHandler{
		DB:       newTestDB(t),
		JWT:      newTestJWT(t),
		Provider: provider,
		Config: oauth2.Config{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  srv.URL + "/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  srv.URL + "/authorize",
				TokenURL: srv.URL + "/token",
			},
			Scopes: []string{"openid", "email", "profile"},
		},
		SecureCookies: false,
		linkNonces:    make(map[string]linkNonce),
	}
}

// ---------------------------------------------------------------------------
// Callback – request validation (no token exchange required)
// ---------------------------------------------------------------------------

func TestOIDCCallback_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/callback", nil)
	w := httptest.NewRecorder()
	h.Callback(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestOIDCCallback_MissingStateCookie(t *testing.T) {
	t.Parallel()
	h := newTestOIDCHandler(t)

	// No cookies attached — state cookie is absent.
	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/callback?state=abc&code=xyz", nil)
	w := httptest.NewRecorder()
	h.Callback(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOIDCCallback_StateMismatch(t *testing.T) {
	t.Parallel()
	h := newTestOIDCHandler(t)

	// Cookie contains "original-state" but query param has "different-state".
	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/callback?state=different-state&code=xyz", nil)
	r.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: "original-state"})
	w := httptest.NewRecorder()
	h.Callback(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOIDCCallback_MissingVerifierCookie(t *testing.T) {
	t.Parallel()
	h := newTestOIDCHandler(t)

	// State cookie is present and matches but the PKCE verifier cookie is absent.
	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/callback?state=test-state&code=xyz", nil)
	r.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: "test-state"})
	w := httptest.NewRecorder()
	h.Callback(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOIDCCallback_ProviderError(t *testing.T) {
	t.Parallel()
	h := newTestOIDCHandler(t)

	// The OIDC provider appended an error parameter to the redirect URI.
	r := httptest.NewRequest(http.MethodGet,
		"/auth/oidc/callback?state=test-state&error=access_denied&error_description=user+denied", nil)
	r.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: "test-state"})
	r.AddCookie(&http.Cookie{Name: oidcVerifierCookieName, Value: "test-verifier"})
	w := httptest.NewRecorder()
	h.Callback(w, r)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOIDCCallback_MissingCode(t *testing.T) {
	t.Parallel()
	h := newTestOIDCHandler(t)

	// State and verifier cookies are present, but the authorization code is absent.
	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/callback?state=test-state", nil)
	r.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: "test-state"})
	r.AddCookie(&http.Cookie{Name: oidcVerifierCookieName, Value: "test-verifier"})
	w := httptest.NewRecorder()
	h.Callback(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Callback – full flow via mock OIDC provider
// ---------------------------------------------------------------------------

func TestOIDCCallback_MissingIDToken(t *testing.T) {
	// Token endpoint returns a valid OAuth2 response but omits the id_token field.
	h := newTestOIDCHandlerWithTokenResponse(t, map[string]any{
		"access_token": "fake-access-token",
		"token_type":   "Bearer",
		// id_token intentionally omitted
	})

	r := callbackRequest("test-state")
	w := httptest.NewRecorder()
	h.Callback(w, r)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body), "decode body")
	require.Equal(t, "missing id_token in response", body["error"])
}

func TestOIDCCallback_MissingSub(t *testing.T) {
	tp := newTestOIDCProvider(t)

	// Claims without a sub field — the zero value "" triggers the validation check.
	tp.setClaims(map[string]any{
		"email":          "nosub@example.com",
		"name":           "No Sub",
		"email_verified": true,
	})

	w := httptest.NewRecorder()
	tp.handler.Callback(w, callbackRequest("test-state"))

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body), "decode body")
	require.Equal(t, "sub claim is required", body["error"])
}

func TestOIDCCallback_MissingEmail(t *testing.T) {
	tp := newTestOIDCProvider(t)

	// Claims without an email field — the zero value "" triggers the validation check.
	tp.setClaims(map[string]any{
		"sub":            "noemail-sub",
		"name":           "No Email",
		"email_verified": true,
	})

	w := httptest.NewRecorder()
	tp.handler.Callback(w, callbackRequest("test-state"))

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body), "decode body")
	require.Equal(t, "email claim is required", body["error"])
}

func TestOIDCCallback_NameFallsBackToEmail(t *testing.T) {
	tp := newTestOIDCProvider(t)

	// When the name claim is absent, the handler substitutes the email address.
	tp.setClaims(map[string]any{
		"sub":            "name-fallback-sub",
		"email":          "namefallback@example.com",
		"email_verified": true,
		// name intentionally omitted
	})

	w := httptest.NewRecorder()
	tp.handler.Callback(w, callbackRequest("test-state"))

	resp := w.Result()
	require.Equal(t, http.StatusFound, resp.StatusCode)

	// Verify the created user has the email as their display name.
	user, err := tp.handler.DB.GetUserByOIDCSubject(t.Context(), "name-fallback-sub")
	require.NoError(t, err, "GetUserByOIDCSubject")
	require.Equal(t, "namefallback@example.com", user.Name)
}

func TestOIDCCallback_LinkFlow_UserNotFound(t *testing.T) {
	tp := newTestOIDCProvider(t)

	tp.setClaims(map[string]any{
		"sub":            "link-sub-notfound",
		"email":          "notfound@example.com",
		"email_verified": false,
	})

	// Sign state referencing a user ID that does not exist in the DB.
	signedState := tp.handler.signLinkState("random-state", "non-existent-user-id")

	w := httptest.NewRecorder()
	tp.handler.Callback(w, callbackRequest(signedState))

	resp := w.Result()
	require.Equal(t, http.StatusFound, resp.StatusCode)
	loc := resp.Header.Get("Location")
	require.Contains(t, loc, "oidc_link_error=")
	require.Contains(t, loc, "User+not+found")
}

func TestOIDCCallback_LinkFlow_AlreadyLinked(t *testing.T) {
	tp := newTestOIDCProvider(t)

	// Create a user that already has an OIDC subject linked.
	user, err := tp.handler.DB.CreateOIDCUser(t.Context(), "Already Linked", "alreadylinked@example.com", "existing-sub")
	require.NoError(t, err, "CreateOIDCUser")

	tp.setClaims(map[string]any{
		"sub":            "new-sub-for-linked-user",
		"email":          "alreadylinked@example.com",
		"email_verified": false,
	})

	// Sign state referencing the already-linked user.
	signedState := tp.handler.signLinkState("random-state", user.ID)

	w := httptest.NewRecorder()
	tp.handler.Callback(w, callbackRequest(signedState))

	resp := w.Result()
	require.Equal(t, http.StatusFound, resp.StatusCode)
	loc := resp.Header.Get("Location")
	require.Contains(t, loc, "oidc_link_error=")
	require.Contains(t, loc, "already+linked")
}

func TestOIDCCallback_LinkFlow_SubjectAlreadyLinkedToOther(t *testing.T) {
	tp := newTestOIDCProvider(t)

	// Create user A (the intended link target, with no OIDC subject).
	userA, err := tp.handler.DB.CreateUser(t.Context(), "User A", "usera@example.com", "password123")
	require.NoError(t, err, "CreateUser (A)")

	// Create user B who already owns the OIDC subject we will claim in the callback.
	_, err = tp.handler.DB.CreateOIDCUser(t.Context(), "User B", "userb@example.com", "taken-sub")
	require.NoError(t, err, "CreateOIDCUser (B)")

	// The callback presents "taken-sub", which belongs to user B, not user A.
	tp.setClaims(map[string]any{
		"sub":            "taken-sub",
		"email":          "usera@example.com",
		"email_verified": false,
	})

	// Sign state referencing user A as the link target.
	signedState := tp.handler.signLinkState("random-state", userA.ID)

	w := httptest.NewRecorder()
	tp.handler.Callback(w, callbackRequest(signedState))

	resp := w.Result()
	require.Equal(t, http.StatusFound, resp.StatusCode)
	loc := resp.Header.Get("Location")
	require.Contains(t, loc, "oidc_link_error=")
	require.Contains(t, loc, "already+linked+to+another")
}

func TestOIDCCallback_Success_SetsCookie(t *testing.T) {
	tp := newTestOIDCProvider(t)

	tp.setClaims(map[string]any{
		"sub":            "cookie-test-sub",
		"email":          "cookietest@example.com",
		"name":           "Cookie Test User",
		"email_verified": true,
	})

	w := httptest.NewRecorder()
	tp.handler.Callback(w, callbackRequest("test-state"))

	resp := w.Result()
	require.Equal(t, http.StatusFound, resp.StatusCode)
	loc := resp.Header.Get("Location")
	require.Equal(t, "/?oidc_login=1", loc)

	// Verify the auth cookie is set with a non-empty token value.
	var authCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == auth.TokenCookieName() {
			authCookie = c
			break
		}
	}
	require.NotNil(t, authCookie)
	require.NotEqual(t, "", authCookie.Value)
	require.True(t, authCookie.HttpOnly)
}

// ---------------------------------------------------------------------------
// findOrCreateUser – direct unit tests
// ---------------------------------------------------------------------------

func TestFindOrCreateUser_FoundBySubject(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Pre-create a user linked to a known OIDC subject.
	existing, err := h.DB.CreateOIDCUser(t.Context(), "Subject User", "subject@example.com", "known-sub")
	require.NoError(t, err, "CreateOIDCUser")

	user, err := h.findOrCreateUser(t.Context(), "known-sub", "subject@example.com", "Subject User")
	require.NoError(t, err, "findOrCreateUser")
	require.Equal(t, existing.ID, user.ID)
}

func TestFindOrCreateUser_FoundByEmail_LinksSubject(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Pre-create a regular (non-OIDC) user.
	existing, err := h.DB.CreateUser(t.Context(), "Email User", "emailmatch@example.com", "password123")
	require.NoError(t, err, "CreateUser")

	// findOrCreateUser should find the user by email and link the new OIDC subject.
	user, err := h.findOrCreateUser(t.Context(), "new-oidc-sub", "emailmatch@example.com", "Email User")
	require.NoError(t, err, "findOrCreateUser")
	require.Equal(t, existing.ID, user.ID)

	// The OIDC subject should now be linked to the existing user.
	linked, err := h.DB.GetUserByOIDCSubject(t.Context(), "new-oidc-sub")
	require.NoError(t, err, "GetUserByOIDCSubject after link")
	require.Equal(t, existing.ID, linked.ID)
}

func TestFindOrCreateUser_CreatesNewUser(t *testing.T) {
	h := newTestOIDCHandler(t)

	user, err := h.findOrCreateUser(t.Context(), "brand-new-sub", "brandnew@example.com", "Brand New User")
	require.NoError(t, err, "findOrCreateUser")

	require.Equal(t, "brandnew@example.com", user.Email)
	require.Equal(t, "Brand New User", user.Name)

	// Confirm the user is retrievable from the DB by OIDC subject.
	found, err := h.DB.GetUserByOIDCSubject(t.Context(), "brand-new-sub")
	require.NoError(t, err, "GetUserByOIDCSubject")
	require.Equal(t, user.ID, found.ID)
}
