package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	if body["error"] != "missing id_token in response" {
		t.Errorf("unexpected error message: %q", body["error"])
	}
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
	if body["error"] != "sub claim is required" {
		t.Errorf("unexpected error message: %q", body["error"])
	}
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
	if body["error"] != "email claim is required" {
		t.Errorf("unexpected error message: %q", body["error"])
	}
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
	if user.Name != "namefallback@example.com" {
		t.Errorf("expected user name %q (email fallback), got %q", "namefallback@example.com", user.Name)
	}
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
	if !strings.Contains(loc, "oidc_link_error=") {
		t.Errorf("expected oidc_link_error in redirect location, got %q", loc)
	}
	if !strings.Contains(loc, "User+not+found") {
		t.Errorf("expected 'User not found' in redirect location, got %q", loc)
	}
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
	if !strings.Contains(loc, "oidc_link_error=") {
		t.Errorf("expected oidc_link_error in redirect location, got %q", loc)
	}
	if !strings.Contains(loc, "already+linked") {
		t.Errorf("expected 'already linked' in redirect location, got %q", loc)
	}
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
	if !strings.Contains(loc, "oidc_link_error=") {
		t.Errorf("expected oidc_link_error in redirect location, got %q", loc)
	}
	if !strings.Contains(loc, "already+linked+to+another") {
		t.Errorf("expected 'already linked to another' in redirect location, got %q", loc)
	}
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
	if loc := resp.Header.Get("Location"); loc != "/?oidc_login=1" {
		t.Errorf("expected Location /?oidc_login=1, got %q", loc)
	}

	// Verify the auth cookie is set with a non-empty token value.
	var authCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == auth.TokenCookieName() {
			authCookie = c
			break
		}
	}
	require.NotNil(t, authCookie)
	if authCookie.Value == "" {
		t.Error("auth cookie value should be non-empty")
	}
	if !authCookie.HttpOnly {
		t.Error("auth cookie should be HttpOnly")
	}
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
	if user.ID != existing.ID {
		t.Errorf("expected user ID %q (found by subject), got %q", existing.ID, user.ID)
	}
}

func TestFindOrCreateUser_FoundByEmail_LinksSubject(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Pre-create a regular (non-OIDC) user.
	existing, err := h.DB.CreateUser(t.Context(), "Email User", "emailmatch@example.com", "password123")
	require.NoError(t, err, "CreateUser")

	// findOrCreateUser should find the user by email and link the new OIDC subject.
	user, err := h.findOrCreateUser(t.Context(), "new-oidc-sub", "emailmatch@example.com", "Email User")
	require.NoError(t, err, "findOrCreateUser")
	if user.ID != existing.ID {
		t.Errorf("expected user ID %q (found by email), got %q", existing.ID, user.ID)
	}

	// The OIDC subject should now be linked to the existing user.
	linked, err := h.DB.GetUserByOIDCSubject(t.Context(), "new-oidc-sub")
	require.NoError(t, err, "GetUserByOIDCSubject after link")
	if linked.ID != existing.ID {
		t.Errorf("OIDC subject not linked to the correct user: expected %q, got %q", existing.ID, linked.ID)
	}
}

func TestFindOrCreateUser_CreatesNewUser(t *testing.T) {
	h := newTestOIDCHandler(t)

	user, err := h.findOrCreateUser(t.Context(), "brand-new-sub", "brandnew@example.com", "Brand New User")
	require.NoError(t, err, "findOrCreateUser")

	if user.Email != "brandnew@example.com" {
		t.Errorf("expected email %q, got %q", "brandnew@example.com", user.Email)
	}
	if user.Name != "Brand New User" {
		t.Errorf("expected name %q, got %q", "Brand New User", user.Name)
	}

	// Confirm the user is retrievable from the DB by OIDC subject.
	found, err := h.DB.GetUserByOIDCSubject(t.Context(), "brand-new-sub")
	require.NoError(t, err, "GetUserByOIDCSubject")
	if found.ID != user.ID {
		t.Errorf("DB lookup returned ID %q, expected %q", found.ID, user.ID)
	}
}
