package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/oauth2"

	"github.com/stretchr/testify/require"
)

func newTestOIDCHandler(t *testing.T) *OIDCHandler {
	t.Helper()
	d := newTestDB(t)
	return &OIDCHandler{
		DB:  d,
		JWT: newTestJWT(t),
		Config: oauth2.Config{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "http://localhost/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "http://fake-provider/authorize",
				TokenURL: "http://fake-provider/token",
			},
			Scopes: []string{"openid", "email", "profile"},
		},
		SecureCookies: false,
		linkNonces:    make(map[string]linkNonce),
	}
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestOIDCLogin_Success(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/login", nil)
	w := httptest.NewRecorder()
	h.Login(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		require.Failf(t, "failed", "expected status 302, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		require.Fail(t, "expected Location header to be set")
	}

	var foundState, foundVerifier bool
	for _, c := range resp.Cookies() {
		switch c.Name {
		case oidcStateCookieName:
			foundState = true
			if c.Value == "" {
				t.Error("state cookie value is empty")
			}
			if !c.HttpOnly {
				t.Error("state cookie should be HttpOnly")
			}
		case oidcVerifierCookieName:
			foundVerifier = true
			if c.Value == "" {
				t.Error("verifier cookie value is empty")
			}
			if !c.HttpOnly {
				t.Error("verifier cookie should be HttpOnly")
			}
		}
	}
	if !foundState {
		t.Error("state cookie not set")
	}
	if !foundVerifier {
		t.Error("verifier cookie not set")
	}
}

func TestOIDCLogin_MethodNotAllowed(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/login", nil)
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		require.Failf(t, "failed", "expected status 405, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// CreateLinkNonce
// ---------------------------------------------------------------------------

func TestOIDCCreateLinkNonce_Success(t *testing.T) {
	h := newTestOIDCHandler(t)

	user, err := h.DB.CreateUser(t.Context(), "Alice", "alice@example.com", "password123")
	require.NoError(t, err, "CreateUser")

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/link-nonce", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()
	h.CreateLinkNonce(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "expected status 200, got %d", w.Code)
	}

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body), "decode response")
	if body["nonce"] == "" {
		require.Fail(t, "expected non-empty nonce in response")
	}
}

func TestOIDCCreateLinkNonce_MethodNotAllowed(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link-nonce", nil)
	w := httptest.NewRecorder()
	h.CreateLinkNonce(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		require.Failf(t, "failed", "expected status 405, got %d", w.Code)
	}
}

func TestOIDCCreateLinkNonce_StoresNonce(t *testing.T) {
	h := newTestOIDCHandler(t)

	user, err := h.DB.CreateUser(t.Context(), "Bob", "bob@example.com", "password123")
	require.NoError(t, err, "CreateUser")

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/link-nonce", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()
	h.CreateLinkNonce(w, r)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body), "decode response")
	nonce := body["nonce"]

	h.linkNoncesMu.Lock()
	entry, ok := h.linkNonces[nonce]
	h.linkNoncesMu.Unlock()

	if !ok {
		require.Fail(t, "nonce not found in linkNonces map")
	}
	if entry.UserID != user.ID {
		t.Errorf("expected UserID %q, got %q", user.ID, entry.UserID)
	}
	if time.Until(entry.ExpiresAt) <= 0 {
		t.Error("nonce should not already be expired")
	}
}

// ---------------------------------------------------------------------------
// consumeLinkNonce
// ---------------------------------------------------------------------------

func TestOIDCConsumeLinkNonce_Valid(t *testing.T) {
	h := newTestOIDCHandler(t)

	h.linkNoncesMu.Lock()
	h.linkNonces["valid-nonce"] = linkNonce{
		UserID:    "user-123",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	h.linkNoncesMu.Unlock()

	got := h.consumeLinkNonce("valid-nonce")
	if got != "user-123" {
		require.Failf(t, "failed", "expected user ID %q, got %q", "user-123", got)
	}
}

func TestOIDCConsumeLinkNonce_InvalidNonce(t *testing.T) {
	h := newTestOIDCHandler(t)

	got := h.consumeLinkNonce("does-not-exist")
	if got != "" {
		require.Failf(t, "failed", "expected empty string for invalid nonce, got %q", got)
	}
}

func TestOIDCConsumeLinkNonce_ExpiredNonce(t *testing.T) {
	h := newTestOIDCHandler(t)

	h.linkNoncesMu.Lock()
	h.linkNonces["expired-nonce"] = linkNonce{
		UserID:    "user-456",
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	h.linkNoncesMu.Unlock()

	got := h.consumeLinkNonce("expired-nonce")
	if got != "" {
		require.Failf(t, "failed", "expected empty string for expired nonce, got %q", got)
	}

	// Verify the nonce was removed even though it was expired
	h.linkNoncesMu.Lock()
	_, exists := h.linkNonces["expired-nonce"]
	h.linkNoncesMu.Unlock()
	if exists {
		t.Error("expired nonce should have been removed from the map")
	}
}

func TestOIDCConsumeLinkNonce_DoubleConsume(t *testing.T) {
	h := newTestOIDCHandler(t)

	h.linkNoncesMu.Lock()
	h.linkNonces["once-only"] = linkNonce{
		UserID:    "user-789",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	h.linkNoncesMu.Unlock()

	first := h.consumeLinkNonce("once-only")
	if first != "user-789" {
		require.Failf(t, "failed", "first consume: expected %q, got %q", "user-789", first)
	}

	second := h.consumeLinkNonce("once-only")
	if second != "" {
		require.Failf(t, "failed", "second consume: expected empty string, got %q", second)
	}
}

// ---------------------------------------------------------------------------
// Link
// ---------------------------------------------------------------------------

func TestOIDCLink_MissingNonce(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link", nil)
	w := httptest.NewRecorder()
	h.Link(w, r)

	if w.Code != http.StatusBadRequest {
		require.Failf(t, "failed", "expected status 400, got %d", w.Code)
	}
}

func TestOIDCLink_InvalidNonce(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link?nonce=bad-nonce", nil)
	w := httptest.NewRecorder()
	h.Link(w, r)

	if w.Code != http.StatusUnauthorized {
		require.Failf(t, "failed", "expected status 401, got %d", w.Code)
	}
}

func TestOIDCLink_MethodNotAllowed(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/link?nonce=something", nil)
	w := httptest.NewRecorder()
	h.Link(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		require.Failf(t, "failed", "expected status 405, got %d", w.Code)
	}
}

func TestOIDCLink_AlreadyLinked(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Create a user that already has an OIDC subject linked
	user, err := h.DB.CreateOIDCUser(t.Context(), "Linked User", "linked@example.com", "existing-subject")
	require.NoError(t, err, "CreateOIDCUser")

	// Seed a valid nonce for this user
	h.linkNoncesMu.Lock()
	h.linkNonces["linked-nonce"] = linkNonce{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	h.linkNoncesMu.Unlock()

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link?nonce=linked-nonce", nil)
	w := httptest.NewRecorder()
	h.Link(w, r)

	if w.Code != http.StatusConflict {
		require.Failf(t, "failed", "expected status 409, got %d", w.Code)
	}
}

func TestOIDCLink_Success(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Create a regular user (no OIDC subject)
	user, err := h.DB.CreateUser(t.Context(), "Normal User", "normal@example.com", "password123")
	require.NoError(t, err, "CreateUser")

	// Seed a valid nonce for this user
	h.linkNoncesMu.Lock()
	h.linkNonces["good-nonce"] = linkNonce{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	h.linkNoncesMu.Unlock()

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link?nonce=good-nonce", nil)
	w := httptest.NewRecorder()
	h.Link(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		require.Failf(t, "failed", "expected status 302, got %d", resp.StatusCode)
	}

	// Verify cookies are set (state and verifier only — no link user ID cookie)
	var foundState, foundVerifier bool
	var stateValue string
	for _, c := range resp.Cookies() {
		switch c.Name {
		case oidcStateCookieName:
			foundState = true
			stateValue = c.Value
			if c.Value == "" {
				t.Error("state cookie value is empty")
			}
		case oidcVerifierCookieName:
			foundVerifier = true
			if c.Value == "" {
				t.Error("verifier cookie value is empty")
			}
		}
	}
	if !foundState {
		t.Error("state cookie not set")
	}
	if !foundVerifier {
		t.Error("verifier cookie not set")
	}

	// Verify the state cookie contains a signed link state with the user ID
	parsedUserID := h.parseLinkState(stateValue)
	if parsedUserID != user.ID {
		t.Errorf("expected parseLinkState to return %q, got %q", user.ID, parsedUserID)
	}

	// Verify nonce was consumed
	h.linkNoncesMu.Lock()
	_, exists := h.linkNonces["good-nonce"]
	h.linkNoncesMu.Unlock()
	if exists {
		t.Error("nonce should have been consumed")
	}
}

// ---------------------------------------------------------------------------
// consumeLinkNonce – concurrency safety
// ---------------------------------------------------------------------------

func TestOIDCConsumeLinkNonce_Concurrent(t *testing.T) {
	h := newTestOIDCHandler(t)

	h.linkNoncesMu.Lock()
	h.linkNonces["race-nonce"] = linkNonce{
		UserID:    "user-race",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	h.linkNoncesMu.Unlock()

	const goroutines = 10
	results := make(chan string, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			results <- h.consumeLinkNonce("race-nonce")
		}()
	}
	wg.Wait()
	close(results)

	var winners int
	for r := range results {
		if r == "user-race" {
			winners++
		}
	}
	if winners != 1 {
		require.Failf(t, "failed", "expected exactly 1 winner, got %d", winners)
	}
}

// ---------------------------------------------------------------------------
// signLinkState / parseLinkState
// ---------------------------------------------------------------------------

func TestOIDCParseLinkState_Valid(t *testing.T) {
	h := newTestOIDCHandler(t)

	signed := h.signLinkState("random-state", "user-123")
	got := h.parseLinkState(signed)
	if got != "user-123" {
		require.Failf(t, "failed", "expected user ID %q, got %q", "user-123", got)
	}
}

func TestOIDCParseLinkState_NormalLogin(t *testing.T) {
	h := newTestOIDCHandler(t)

	// A plain state (no dots) should return empty — this is a normal login.
	got := h.parseLinkState("plain-state-no-dots")
	if got != "" {
		require.Failf(t, "failed", "expected empty string for normal login state, got %q", got)
	}
}

func TestOIDCParseLinkState_ManualTamperUserID(t *testing.T) {
	h := newTestOIDCHandler(t)

	signed := h.signLinkState("random-state", "user-123")
	parts := strings.SplitN(signed, ".", 3)
	// Replace userID with a different one, leave HMAC unchanged
	parts[1] = base64.RawURLEncoding.EncodeToString([]byte("victim-user"))
	tampered := strings.Join(parts, ".")

	if got := h.parseLinkState(tampered); got != "" {
		require.Failf(t, "failed", "expected empty string for tampered state, got %q", got)
	}
}

func TestOIDCParseLinkState_InvalidSignature(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Construct a state with a bad HMAC
	got := h.parseLinkState("random-state.dXNlci0xMjM.bm90LWEtdmFsaWQtc2ln")
	if got != "" {
		require.Failf(t, "failed", "expected empty string for invalid signature, got %q", got)
	}
}

func TestOIDCParseLinkState_DifferentSecret(t *testing.T) {
	h1 := newTestOIDCHandler(t)

	// Create a handler with a different JWT secret
	differentJWT, err := auth.NewJWTManager("different-secret-key", time.Hour)
	require.NoError(t, err, "NewJWTManager")
	h2 := newTestOIDCHandler(t)
	h2.JWT = differentJWT

	signed := h1.signLinkState("random-state", "user-123")
	// h2 has a different JWT secret, so it should reject h1's signature
	got := h2.parseLinkState(signed)
	if got != "" {
		require.Failf(t, "failed", "expected empty string when verifying with different secret, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Callback – email_verified enforcement
// ---------------------------------------------------------------------------

// testOIDCProvider spins up a mock OIDC provider (discovery + JWKS + token
// exchange) and returns an OIDCHandler wired to it. The setClaims function
// controls the claims embedded in the ID token returned by the /token endpoint.
type testOIDCProvider struct {
	handler   *OIDCHandler
	setClaims func(claims map[string]any)
	server    *httptest.Server
}

func newTestOIDCProvider(t *testing.T) *testOIDCProvider {
	t.Helper()

	// Generate an RSA key pair for signing ID tokens.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "generate RSA key")

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: rsaKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"),
	)
	require.NoError(t, err, "create signer")

	jwks := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{{
			Key:       rsaKey.Public(),
			KeyID:     "test-key",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}},
	}

	// signToken produces a compact JWT from an arbitrary claims map.
	signToken := func(claims map[string]any) string {
		t.Helper()
		builder := josejwt.Signed(signer).Claims(claims)
		raw, err := builder.Serialize()
		require.NoError(t, err, "sign id_token")
		return raw
	}

	// We need a placeholder for the server URL before starting it, because
	// the token endpoint handler needs to produce tokens with the server's
	// own URL as issuer. We solve this by capturing a pointer.
	var serverURL string

	// idTokenClaims is set per-test to control what the token endpoint returns.
	var idTokenClaims map[string]any

	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                serverURL,
			"authorization_endpoint":                serverURL + "/authorize",
			"token_endpoint":                        serverURL + "/token",
			"jwks_uri":                              serverURL + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		// Inject issuer and timing claims if not already present.
		now := time.Now()
		defaults := map[string]any{
			"iss": serverURL,
			"aud": "test-client",
			"iat": now.Unix(),
			"exp": now.Add(time.Hour).Unix(),
		}
		merged := make(map[string]any)
		maps.Copy(merged, defaults)
		maps.Copy(merged, idTokenClaims)

		idToken := signToken(merged)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fake-access-token",
			"token_type":   "Bearer",
			"id_token":     idToken,
		})
	})

	srv := httptest.NewServer(mux)
	serverURL = srv.URL
	t.Cleanup(srv.Close)

	// Create a real OIDC provider pointing at our test server.
	provider, err := oidc.NewProvider(t.Context(), srv.URL)
	require.NoError(t, err, "oidc.NewProvider")

	d := newTestDB(t)
	h := &OIDCHandler{
		DB:       d,
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

	tp := &testOIDCProvider{
		handler: h,
		server:  srv,
		setClaims: func(claims map[string]any) {
			idTokenClaims = claims
		},
	}

	return tp
}

// callbackRequest builds a GET /callback request with valid state/verifier cookies.
func callbackRequest(state string) *http.Request {
	r := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/auth/oidc/callback?state=%s&code=test-code", state), nil)
	r.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
	r.AddCookie(&http.Cookie{Name: oidcVerifierCookieName, Value: "test-verifier"})
	return r
}

func TestOIDCCallback_EmailVerifiedTrue(t *testing.T) {
	tp := newTestOIDCProvider(t)
	h := tp.handler

	tp.setClaims(map[string]any{
		"sub":            "oidc-user-1",
		"email":          "verified@example.com",
		"name":           "Verified User",
		"email_verified": true,
	})

	w := httptest.NewRecorder()
	h.Callback(w, callbackRequest("test-state"))

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		require.Failf(t, "failed", "expected redirect (302), got %d; body: %s", resp.StatusCode, w.Body.String())
	}
	loc := resp.Header.Get("Location")
	if loc != "/?oidc_login=1" {
		t.Errorf("expected redirect to /?oidc_login=1, got %q", loc)
	}
}

func TestOIDCCallback_EmailVerifiedFalse(t *testing.T) {
	tp := newTestOIDCProvider(t)
	h := tp.handler

	tp.setClaims(map[string]any{
		"sub":            "oidc-user-2",
		"email":          "unverified@example.com",
		"name":           "Unverified User",
		"email_verified": false,
	})

	w := httptest.NewRecorder()
	h.Callback(w, callbackRequest("test-state"))

	if w.Code != http.StatusUnauthorized {
		require.Failf(t, "failed", "expected status 401, got %d; body: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body), "decode body")
	if body["error"] != "OIDC email must be verified by the identity provider" {
		t.Errorf("unexpected error message: %q", body["error"])
	}
}

func TestOIDCCallback_EmailVerifiedMissing(t *testing.T) {
	tp := newTestOIDCProvider(t)
	h := tp.handler

	tp.setClaims(map[string]any{
		"sub":   "oidc-user-3",
		"email": "noverify@example.com",
		"name":  "No Verify User",
		// email_verified intentionally omitted
	})

	w := httptest.NewRecorder()
	h.Callback(w, callbackRequest("test-state"))

	if w.Code != http.StatusUnauthorized {
		require.Failf(t, "failed", "expected status 401, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestOIDCCallback_LinkFlowBypassesEmailVerified(t *testing.T) {
	tp := newTestOIDCProvider(t)
	h := tp.handler

	// Create a user to link to.
	user, err := h.DB.CreateUser(t.Context(), "Link Target", "linktarget@example.com", "password123")
	require.NoError(t, err, "CreateUser")

	tp.setClaims(map[string]any{
		"sub":            "oidc-link-subject",
		"email":          "linktarget@example.com",
		"name":           "Link Target",
		"email_verified": false,
	})

	// Create a signed link state (simulates what Link() does)
	signedState := h.signLinkState("test-state", user.ID)

	w := httptest.NewRecorder()
	h.Callback(w, callbackRequest(signedState))

	resp := w.Result()
	// Link flow should succeed (redirect) even without email_verified.
	if resp.StatusCode != http.StatusFound {
		require.Failf(t, "failed", "expected redirect (302), got %d; body: %s", resp.StatusCode, w.Body.String())
	}
	loc := resp.Header.Get("Location")
	if loc != "/?oidc_linked=true" {
		t.Errorf("expected redirect to /?oidc_linked=true, got %q", loc)
	}
}
