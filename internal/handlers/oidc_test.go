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

// seedLinkNonce inserts a nonce entry into h.linkNonces under key for userID with
// the given expiry offset. Use a negative duration to create an already-expired
// nonce.
func seedLinkNonce(h *OIDCHandler, key, userID string, expiry time.Duration) {
	h.linkNoncesMu.Lock()
	h.linkNonces[key] = linkNonce{UserID: userID, ExpiresAt: time.Now().Add(expiry)}
	h.linkNoncesMu.Unlock()
}

// requireOIDCCookies asserts that cookies contains non-empty state and verifier
// OIDC cookies and returns the state cookie value.
func requireOIDCCookies(t *testing.T, cookies []*http.Cookie) string {
	t.Helper()
	var stateVal string
	var foundState, foundVerifier bool
	for _, c := range cookies {
		switch c.Name {
		case oidcStateCookieName:
			foundState = true
			stateVal = c.Value
			require.NotEmpty(t, c.Value, "state cookie value is empty")
		case oidcVerifierCookieName:
			foundVerifier = true
			require.NotEmpty(t, c.Value, "verifier cookie value is empty")
		}
	}
	require.True(t, foundState, "state cookie not set")
	require.True(t, foundVerifier, "verifier cookie not set")
	return stateVal
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
	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.NotEmpty(t, resp.Header.Get("Location"))

	cookies := resp.Cookies()
	requireOIDCCookies(t, cookies)
	for _, c := range cookies {
		if c.Name == oidcStateCookieName || c.Name == oidcVerifierCookieName {
			require.True(t, c.HttpOnly, "%s cookie should be HttpOnly", c.Name)
		}
	}
}

func TestOIDCLogin_MethodNotAllowed(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/login", nil)
	w := httptest.NewRecorder()
	h.Login(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
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

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body), "decode response")
	require.NotEmpty(t, body["nonce"])
}

func TestOIDCCreateLinkNonce_MethodNotAllowed(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link-nonce", nil)
	w := httptest.NewRecorder()
	h.CreateLinkNonce(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
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

	require.True(t, ok)
	require.Equal(t, user.ID, entry.UserID)
	require.True(t, time.Until(entry.ExpiresAt) > 0, "nonce should not already be expired")
}

// ---------------------------------------------------------------------------
// consumeLinkNonce
// ---------------------------------------------------------------------------

func TestOIDCConsumeLinkNonce(t *testing.T) {
	tests := []struct {
		name           string
		nonceKey       string
		seed           bool
		expiry         time.Duration
		userID         string
		wantUserID     string
		wantKeyRemoved bool
	}{
		{
			name:           "valid nonce returns user ID",
			nonceKey:       "valid-nonce",
			seed:           true,
			expiry:         5 * time.Minute,
			userID:         "user-123",
			wantUserID:     "user-123",
			wantKeyRemoved: true,
		},
		{
			name:       "unknown nonce returns empty",
			nonceKey:   "does-not-exist",
			wantUserID: "",
		},
		{
			name:           "expired nonce returns empty and is removed",
			nonceKey:       "expired-nonce",
			seed:           true,
			expiry:         -1 * time.Minute,
			userID:         "user-456",
			wantUserID:     "",
			wantKeyRemoved: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestOIDCHandler(t)
			if tt.seed {
				seedLinkNonce(h, tt.nonceKey, tt.userID, tt.expiry)
			}
			got := h.consumeLinkNonce(tt.nonceKey)
			require.Equal(t, tt.wantUserID, got)
			if tt.wantKeyRemoved {
				h.linkNoncesMu.Lock()
				_, exists := h.linkNonces[tt.nonceKey]
				h.linkNoncesMu.Unlock()
				require.False(t, exists, "expired nonce should have been removed from the map")
			}
		})
	}
}

func TestOIDCConsumeLinkNonce_DoubleConsume(t *testing.T) {
	h := newTestOIDCHandler(t)
	seedLinkNonce(h, "once-only", "user-789", 5*time.Minute)

	first := h.consumeLinkNonce("once-only")
	require.Equal(t, "user-789", first)

	second := h.consumeLinkNonce("once-only")
	require.Equal(t, "", second)
}

// ---------------------------------------------------------------------------
// consumeLinkNonce – concurrency safety
// ---------------------------------------------------------------------------

func TestOIDCConsumeLinkNonce_Concurrent(t *testing.T) {
	h := newTestOIDCHandler(t)
	seedLinkNonce(h, "race-nonce", "user-race", 5*time.Minute)

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
	require.Equal(t, 1, winners)
}

// ---------------------------------------------------------------------------
// Link
// ---------------------------------------------------------------------------

func TestOIDCLink_Errors(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		url      string
		wantCode int
	}{
		{name: "method not allowed", method: http.MethodPost, url: "/auth/oidc/link?nonce=something", wantCode: http.StatusMethodNotAllowed},
		{name: "missing nonce", method: http.MethodGet, url: "/auth/oidc/link", wantCode: http.StatusBadRequest},
		{name: "invalid nonce", method: http.MethodGet, url: "/auth/oidc/link?nonce=bad-nonce", wantCode: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestOIDCHandler(t)
			r := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			h.Link(w, r)
			require.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestOIDCLink_AlreadyLinked(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Create a user that already has an OIDC subject linked.
	user, err := h.DB.CreateOIDCUser(t.Context(), "Linked User", "linked@example.com", "existing-subject")
	require.NoError(t, err, "CreateOIDCUser")

	seedLinkNonce(h, "linked-nonce", user.ID, 5*time.Minute)

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link?nonce=linked-nonce", nil)
	w := httptest.NewRecorder()
	h.Link(w, r)

	require.Equal(t, http.StatusConflict, w.Code)
}

func TestOIDCLink_Success(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Create a regular user (no OIDC subject).
	user, err := h.DB.CreateUser(t.Context(), "Normal User", "normal@example.com", "password123")
	require.NoError(t, err, "CreateUser")

	seedLinkNonce(h, "good-nonce", user.ID, 5*time.Minute)

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link?nonce=good-nonce", nil)
	w := httptest.NewRecorder()
	h.Link(w, r)

	resp := w.Result()
	require.Equal(t, http.StatusFound, resp.StatusCode)

	// Verify cookies are set (state and verifier only — no link user ID cookie).
	stateValue := requireOIDCCookies(t, resp.Cookies())

	// Verify the state cookie contains a signed link state with the user ID.
	parsedUserID := h.parseLinkState(stateValue)
	require.Equal(t, user.ID, parsedUserID)

	// Verify nonce was consumed.
	h.linkNoncesMu.Lock()
	_, exists := h.linkNonces["good-nonce"]
	h.linkNoncesMu.Unlock()
	require.False(t, exists, "nonce should have been consumed")
}

// ---------------------------------------------------------------------------
// signLinkState / parseLinkState
// ---------------------------------------------------------------------------

func TestOIDCParseLinkState(t *testing.T) {
	tests := []struct {
		name  string
		setup func(h *OIDCHandler) string
		want  string
	}{
		{
			name:  "valid signed state returns user ID",
			setup: func(h *OIDCHandler) string { return h.signLinkState("random-state", "user-123") },
			want:  "user-123",
		},
		{
			// A plain state (no dots) should return empty — this is a normal login.
			name:  "plain state without dots is a normal login",
			setup: func(_ *OIDCHandler) string { return "plain-state-no-dots" },
			want:  "",
		},
		{
			name: "tampered user ID is rejected",
			setup: func(h *OIDCHandler) string {
				signed := h.signLinkState("random-state", "user-123")
				parts := strings.SplitN(signed, ".", 3)
				// Replace userID with a different one, leave HMAC unchanged.
				parts[1] = base64.RawURLEncoding.EncodeToString([]byte("victim-user"))
				return strings.Join(parts, ".")
			},
			want: "",
		},
		{
			name:  "invalid HMAC signature is rejected",
			setup: func(_ *OIDCHandler) string { return "random-state.dXNlci0xMjM.bm90LWEtdmFsaWQtc2ln" },
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestOIDCHandler(t)
			got := h.parseLinkState(tt.setup(h))
			require.Equal(t, tt.want, got)
		})
	}
}

func TestOIDCParseLinkState_DifferentSecret(t *testing.T) {
	h1 := newTestOIDCHandler(t)

	// Create a handler with a different JWT secret; it should reject h1's signature.
	differentJWT, err := auth.NewJWTManager("different-secret-key", time.Hour)
	require.NoError(t, err, "NewJWTManager")
	h2 := newTestOIDCHandler(t)
	h2.JWT = differentJWT

	signed := h1.signLinkState("random-state", "user-123")
	got := h2.parseLinkState(signed)
	require.Equal(t, "", got)
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

	return &testOIDCProvider{
		handler: h,
		server:  srv,
		setClaims: func(claims map[string]any) {
			idTokenClaims = claims
		},
	}
}

// callbackRequest builds a GET /callback request with valid state/verifier cookies.
func callbackRequest(state string) *http.Request {
	r := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/auth/oidc/callback?state=%s&code=test-code", state), nil)
	r.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
	r.AddCookie(&http.Cookie{Name: oidcVerifierCookieName, Value: "test-verifier"})
	return r
}

func TestOIDCCallback_EmailVerification(t *testing.T) {
	tests := []struct {
		name         string
		claims       map[string]any
		wantCode     int
		wantLocation string
		wantErrMsg   string
	}{
		{
			name: "verified email redirects to home",
			claims: map[string]any{
				"sub": "oidc-user-1", "email": "verified@example.com",
				"name": "Verified User", "email_verified": true,
			},
			wantCode:     http.StatusFound,
			wantLocation: "/?oidc_login=1",
		},
		{
			name: "unverified email is rejected",
			claims: map[string]any{
				"sub": "oidc-user-2", "email": "unverified@example.com",
				"name": "Unverified User", "email_verified": false,
			},
			wantCode:   http.StatusUnauthorized,
			wantErrMsg: "OIDC email must be verified by the identity provider",
		},
		{
			// email_verified intentionally omitted.
			name: "missing email_verified is rejected",
			claims: map[string]any{
				"sub": "oidc-user-3", "email": "noverify@example.com", "name": "No Verify User",
			},
			wantCode:   http.StatusUnauthorized,
			wantErrMsg: "OIDC email must be verified by the identity provider",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := newTestOIDCProvider(t)
			tp.setClaims(tt.claims)
			w := httptest.NewRecorder()
			tp.handler.Callback(w, callbackRequest("test-state"))
			require.Equal(t, tt.wantCode, w.Code)
			if tt.wantLocation != "" {
				require.Equal(t, tt.wantLocation, w.Result().Header.Get("Location"))
			}
			if tt.wantErrMsg != "" {
				var body map[string]string
				require.NoError(t, json.NewDecoder(w.Body).Decode(&body), "decode body")
				require.Equal(t, tt.wantErrMsg, body["error"])
			}
		})
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

	// Create a signed link state (simulates what Link() does).
	signedState := h.signLinkState("test-state", user.ID)

	w := httptest.NewRecorder()
	h.Callback(w, callbackRequest(signedState))

	resp := w.Result()
	// Link flow should succeed (redirect) even without email_verified.
	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.Equal(t, "/?oidc_linked=true", resp.Header.Get("Location"))
}
