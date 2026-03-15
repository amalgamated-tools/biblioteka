package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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
		t.Fatalf("expected status 302, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatal("expected Location header to be set")
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
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// CreateLinkNonce
// ---------------------------------------------------------------------------

func TestOIDCCreateLinkNonce_Success(t *testing.T) {
	h := newTestOIDCHandler(t)

	user, err := h.DB.CreateUser(context.Background(), "Alice", "alice@example.com", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/link-nonce", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()
	h.CreateLinkNonce(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["nonce"] == "" {
		t.Fatal("expected non-empty nonce in response")
	}
}

func TestOIDCCreateLinkNonce_MethodNotAllowed(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link-nonce", nil)
	w := httptest.NewRecorder()
	h.CreateLinkNonce(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}

func TestOIDCCreateLinkNonce_StoresNonce(t *testing.T) {
	h := newTestOIDCHandler(t)

	user, err := h.DB.CreateUser(context.Background(), "Bob", "bob@example.com", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/link-nonce", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()
	h.CreateLinkNonce(w, r)

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	nonce := body["nonce"]

	h.linkNoncesMu.Lock()
	entry, ok := h.linkNonces[nonce]
	h.linkNoncesMu.Unlock()

	if !ok {
		t.Fatal("nonce not found in linkNonces map")
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
		t.Fatalf("expected user ID %q, got %q", "user-123", got)
	}
}

func TestOIDCConsumeLinkNonce_InvalidNonce(t *testing.T) {
	h := newTestOIDCHandler(t)

	got := h.consumeLinkNonce("does-not-exist")
	if got != "" {
		t.Fatalf("expected empty string for invalid nonce, got %q", got)
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
		t.Fatalf("expected empty string for expired nonce, got %q", got)
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
		t.Fatalf("first consume: expected %q, got %q", "user-789", first)
	}

	second := h.consumeLinkNonce("once-only")
	if second != "" {
		t.Fatalf("second consume: expected empty string, got %q", second)
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
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestOIDCLink_InvalidNonce(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/auth/oidc/link?nonce=bad-nonce", nil)
	w := httptest.NewRecorder()
	h.Link(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestOIDCLink_MethodNotAllowed(t *testing.T) {
	h := newTestOIDCHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/auth/oidc/link?nonce=something", nil)
	w := httptest.NewRecorder()
	h.Link(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}

func TestOIDCLink_AlreadyLinked(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Create a user that already has an OIDC subject linked
	user, err := h.DB.CreateOIDCUser(context.Background(), "Linked User", "linked@example.com", "existing-subject")
	if err != nil {
		t.Fatalf("CreateOIDCUser: %v", err)
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
		t.Fatalf("expected status 409, got %d", w.Code)
	}
}

func TestOIDCLink_Success(t *testing.T) {
	h := newTestOIDCHandler(t)

	// Create a regular user (no OIDC subject)
	user, err := h.DB.CreateUser(context.Background(), "Normal User", "normal@example.com", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
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
		t.Fatalf("expected status 302, got %d", resp.StatusCode)
	}

	// Verify cookies are set
	var foundState, foundVerifier, foundLinkUserID bool
	for _, c := range resp.Cookies() {
		switch c.Name {
		case oidcStateCookieName:
			foundState = true
			if c.Value == "" {
				t.Error("state cookie value is empty")
			}
		case oidcVerifierCookieName:
			foundVerifier = true
			if c.Value == "" {
				t.Error("verifier cookie value is empty")
			}
		case oidcLinkUserIDCookieName:
			foundLinkUserID = true
			if c.Value != user.ID {
				t.Errorf("link user ID cookie: expected %q, got %q", user.ID, c.Value)
			}
		}
	}
	if !foundState {
		t.Error("state cookie not set")
	}
	if !foundVerifier {
		t.Error("verifier cookie not set")
	}
	if !foundLinkUserID {
		t.Error("link user ID cookie not set")
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
		t.Fatalf("expected exactly 1 winner, got %d", winners)
	}
}
