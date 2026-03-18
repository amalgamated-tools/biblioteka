package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/oauth2"
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
		failNowf(t, "expected status 302, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		failNow(t, "expected Location header to be set")
	}

	var foundState, foundVerifier bool
	for _, c := range resp.Cookies() {
		switch c.Name {
		case oidcStateCookieName:
			foundState = true
			if c.Value == "" {
				fail(t, "state cookie value is empty")
			}
			if !c.HttpOnly {
				fail(t, "state cookie should be HttpOnly")
			}
		case oidcVerifierCookieName:
			foundVerifier = true
			if c.Value == "" {
				fail(t, "verifier cookie value is empty")
			}
			if !c.HttpOnly {
				fail(t, "verifier cookie should be HttpOnly")
			}
		}
	}
	if !foundState {
		fail(t, "state cookie not set")
	}
	if !foundVerifier {
		fail(t, "verifier cookie not set")
	}
}

func TestOIDCLogin_MethodNotAllowed(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/login", nil)
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		failNowf(t, "expected status 405, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// CreateLinkNonce
// ---------------------------------------------------------------------------

func TestOIDCCreateLinkNonce_Success(t *testing.T) {
	h := newTestOIDCHandler(t)

	user, err := h.DB.CreateUser(context.Background(), "Alice", "alice@example.com", "password123")
	if err != nil {
		failNowf(t, "CreateUser: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/link-nonce", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()
	h.CreateLinkNonce(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "expected status 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		failNowf(t, "decode response: %v", err)
	}
	if body["nonce"] == "" {
		failNow(t, "expected non-empty nonce in response")
	}
}

func TestOIDCCreateLinkNonce_MethodNotAllowed(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link-nonce", nil)
	w := httptest.NewRecorder()
	h.CreateLinkNonce(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		failNowf(t, "expected status 405, got %d", w.Code)
	}
}

func TestOIDCCreateLinkNonce_StoresNonce(t *testing.T) {
	h := newTestOIDCHandler(t)

	user, err := h.DB.CreateUser(context.Background(), "Bob", "bob@example.com", "password123")
	if err != nil {
		failNowf(t, "CreateUser: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/link-nonce", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()
	h.CreateLinkNonce(w, r)

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		failNowf(t, "decode response: %v", err)
	}
	nonce := body["nonce"]

	h.linkNoncesMu.Lock()
	entry, ok := h.linkNonces[nonce]
	h.linkNoncesMu.Unlock()

	if !ok {
		failNow(t, "nonce not found in linkNonces map")
	}
	if entry.UserID != user.ID {
		failf(t, "expected UserID %q, got %q", user.ID, entry.UserID)
	}
	if time.Until(entry.ExpiresAt) <= 0 {
		fail(t, "nonce should not already be expired")
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
		failNowf(t, "expected user ID %q, got %q", "user-123", got)
	}
}

func TestOIDCConsumeLinkNonce_InvalidNonce(t *testing.T) {
	h := newTestOIDCHandler(t)

	got := h.consumeLinkNonce("does-not-exist")
	if got != "" {
		failNowf(t, "expected empty string for invalid nonce, got %q", got)
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
		failNowf(t, "expected empty string for expired nonce, got %q", got)
	}

	// Verify the nonce was removed even though it was expired
	h.linkNoncesMu.Lock()
	_, exists := h.linkNonces["expired-nonce"]
	h.linkNoncesMu.Unlock()
	if exists {
		fail(t, "expired nonce should have been removed from the map")
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
		failNowf(t, "first consume: expected %q, got %q", "user-789", first)
	}

	second := h.consumeLinkNonce("once-only")
	if second != "" {
		failNowf(t, "second consume: expected empty string, got %q", second)
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
		failNowf(t, "expected status 400, got %d", w.Code)
	}
}

func TestOIDCLink_InvalidNonce(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link?nonce=bad-nonce", nil)
	w := httptest.NewRecorder()
	h.Link(w, r)

	if w.Code != http.StatusUnauthorized {
		failNowf(t, "expected status 401, got %d", w.Code)
	}
}

func TestOIDCLink_MethodNotAllowed(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/link?nonce=something", nil)
	w := httptest.NewRecorder()
	h.Link(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		failNowf(t, "expected status 405, got %d", w.Code)
	}
}

func TestOIDCLink_AlreadyLinked(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Create a user that already has an OIDC subject linked
	user, err := h.DB.CreateOIDCUser(context.Background(), "Linked User", "linked@example.com", "existing-subject")
	if err != nil {
		failNowf(t, "CreateOIDCUser: %v", err)
	}

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
		failNowf(t, "expected status 409, got %d", w.Code)
	}
}

func TestOIDCLink_Success(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Create a regular user (no OIDC subject)
	user, err := h.DB.CreateUser(context.Background(), "Normal User", "normal@example.com", "password123")
	if err != nil {
		failNowf(t, "CreateUser: %v", err)
	}

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
		failNowf(t, "expected status 302, got %d", resp.StatusCode)
	}

	// Verify cookies are set
	var foundState, foundVerifier, foundLinkUserID bool
	for _, c := range resp.Cookies() {
		switch c.Name {
		case oidcStateCookieName:
			foundState = true
			if c.Value == "" {
				fail(t, "state cookie value is empty")
			}
		case oidcVerifierCookieName:
			foundVerifier = true
			if c.Value == "" {
				fail(t, "verifier cookie value is empty")
			}
		case oidcLinkUserIDCookieName:
			foundLinkUserID = true
			if c.Value != user.ID {
				failf(t, "link user ID cookie: expected %q, got %q", user.ID, c.Value)
			}
		}
	}
	if !foundState {
		fail(t, "state cookie not set")
	}
	if !foundVerifier {
		fail(t, "verifier cookie not set")
	}
	if !foundLinkUserID {
		fail(t, "link user ID cookie not set")
	}

	// Verify nonce was consumed
	h.linkNoncesMu.Lock()
	_, exists := h.linkNonces["good-nonce"]
	h.linkNoncesMu.Unlock()
	if exists {
		fail(t, "nonce should have been consumed")
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
		failNowf(t, "expected exactly 1 winner, got %d", winners)
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
	setClaims func(claims map[string]interface{})
	server    *httptest.Server
}

func newTestOIDCProvider(t *testing.T) *testOIDCProvider {
	t.Helper()

	// Generate an RSA key pair for signing ID tokens.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		failNowf(t, "generate RSA key: %v", err)
	}

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: rsaKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"),
	)
	if err != nil {
		failNowf(t, "create signer: %v", err)
	}

	jwks := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{{
			Key:       rsaKey.Public(),
			KeyID:     "test-key",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}},
	}

	// signToken produces a compact JWT from an arbitrary claims map.
	signToken := func(claims map[string]interface{}) string {
		t.Helper()
		builder := josejwt.Signed(signer).Claims(claims)
		raw, err := builder.Serialize()
		if err != nil {
			failNowf(t, "sign id_token: %v", err)
		}
		return raw
	}

	// We need a placeholder for the server URL before starting it, because
	// the token endpoint handler needs to produce tokens with the server's
	// own URL as issuer. We solve this by capturing a pointer.
	var serverURL string

	// idTokenClaims is set per-test to control what the token endpoint returns.
	var idTokenClaims map[string]interface{}

	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
		defaults := map[string]interface{}{
			"iss": serverURL,
			"aud": "test-client",
			"iat": now.Unix(),
			"exp": now.Add(time.Hour).Unix(),
		}
		merged := make(map[string]interface{})
		for k, v := range defaults {
			merged[k] = v
		}
		for k, v := range idTokenClaims {
			merged[k] = v
		}

		idToken := signToken(merged)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
	if err != nil {
		failNowf(t, "oidc.NewProvider: %v", err)
	}

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
		setClaims: func(claims map[string]interface{}) {
			idTokenClaims = claims
		},
	}

	return tp
}

// callbackRequest builds a GET /callback request with valid state/verifier
// cookies and optional link-user-id cookie.
func callbackRequest(state, linkUserID string) *http.Request {
	r := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/auth/oidc/callback?state=%s&code=test-code", state), nil)
	r.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
	r.AddCookie(&http.Cookie{Name: oidcVerifierCookieName, Value: "test-verifier"})
	if linkUserID != "" {
		r.AddCookie(&http.Cookie{Name: oidcLinkUserIDCookieName, Value: linkUserID})
	}
	return r
}

func TestOIDCCallback_EmailVerifiedTrue(t *testing.T) {
	tp := newTestOIDCProvider(t)
	h := tp.handler

	tp.setClaims(map[string]interface{}{
		"sub":            "oidc-user-1",
		"email":          "verified@example.com",
		"name":           "Verified User",
		"email_verified": true,
	})

	w := httptest.NewRecorder()
	h.Callback(w, callbackRequest("test-state", ""))

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		failNowf(t, "expected redirect (302), got %d; body: %s", resp.StatusCode, w.Body.String())
	}
	loc := resp.Header.Get("Location")
	if loc != "/?oidc_login=1" {
		failf(t, "expected redirect to /?oidc_login=1, got %q", loc)
	}
}

func TestOIDCCallback_EmailVerifiedFalse(t *testing.T) {
	tp := newTestOIDCProvider(t)
	h := tp.handler

	tp.setClaims(map[string]interface{}{
		"sub":            "oidc-user-2",
		"email":          "unverified@example.com",
		"name":           "Unverified User",
		"email_verified": false,
	})

	w := httptest.NewRecorder()
	h.Callback(w, callbackRequest("test-state", ""))

	if w.Code != http.StatusUnauthorized {
		failNowf(t, "expected status 401, got %d; body: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		failNowf(t, "decode body: %v", err)
	}
	if body["error"] != "OIDC email must be verified by the identity provider" {
		failf(t, "unexpected error message: %q", body["error"])
	}
}

func TestOIDCCallback_EmailVerifiedMissing(t *testing.T) {
	tp := newTestOIDCProvider(t)
	h := tp.handler

	tp.setClaims(map[string]interface{}{
		"sub":   "oidc-user-3",
		"email": "noverify@example.com",
		"name":  "No Verify User",
		// email_verified intentionally omitted
	})

	w := httptest.NewRecorder()
	h.Callback(w, callbackRequest("test-state", ""))

	if w.Code != http.StatusUnauthorized {
		failNowf(t, "expected status 401, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestOIDCCallback_LinkFlowBypassesEmailVerified(t *testing.T) {
	tp := newTestOIDCProvider(t)
	h := tp.handler

	// Create a user to link to.
	user, err := h.DB.CreateUser(t.Context(), "Link Target", "linktarget@example.com", "password123")
	if err != nil {
		failNowf(t, "CreateUser: %v", err)
	}

	tp.setClaims(map[string]interface{}{
		"sub":            "oidc-link-subject",
		"email":          "linktarget@example.com",
		"name":           "Link Target",
		"email_verified": false,
	})

	w := httptest.NewRecorder()
	h.Callback(w, callbackRequest("test-state", user.ID))

	resp := w.Result()
	// Link flow should succeed (redirect) even without email_verified.
	if resp.StatusCode != http.StatusFound {
		failNowf(t, "expected redirect (302), got %d; body: %s", resp.StatusCode, w.Body.String())
	}
	loc := resp.Header.Get("Location")
	if loc != "/?oidc_linked=true" {
		failf(t, "expected redirect to /?oidc_linked=true, got %q", loc)
	}
}
