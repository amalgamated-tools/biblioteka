package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// seedProviders inserts test watch providers into the database.
func seedProviders(t *testing.T, d *db.DB) {
	t.Helper()
	providers := []db.WatchProvider{
		{ProviderID: 8, ProviderName: "Netflix", LogoPath: "/netflix.jpg", DisplayPriority: 1, ProviderType: "both"},
		{ProviderID: 9, ProviderName: "Amazon Prime Video", LogoPath: "/prime.jpg", DisplayPriority: 2, ProviderType: "both"},
		{ProviderID: 337, ProviderName: "Disney Plus", LogoPath: "/disney.jpg", DisplayPriority: 3, ProviderType: "both"},
	}
	if err := d.UpsertWatchProviders(providers); err != nil {
		t.Fatalf("seed providers: %v", err)
	}
}

func setupWatchProviderHandler(t *testing.T) (*WatchProviderHandler, *db.DB, string) {
	t.Helper()
	d := newTestDB(t)
	h := &WatchProviderHandler{DB: d}

	user, err := d.CreateUser("Alice", "alice@example.com", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	return h, d, user.ID
}

// --- List all providers (GET /api/watch-providers) ---

func TestListWatchProviders_Empty(t *testing.T) {
	h, _, userID := setupWatchProviderHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/watch-providers", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleWatchProviders(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var providers []watchProviderDTO
	if err := json.Unmarshal(w.Body.Bytes(), &providers); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}
}

func TestListWatchProviders_ReturnsSeeded(t *testing.T) {
	h, d, userID := setupWatchProviderHandler(t)
	seedProviders(t, d)

	r := httptest.NewRequest(http.MethodGet, "/api/watch-providers", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleWatchProviders(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var providers []watchProviderDTO
	if err := json.Unmarshal(w.Body.Bytes(), &providers); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(providers))
	}
}

func TestListWatchProviders_MethodNotAllowed(t *testing.T) {
	h, _, userID := setupWatchProviderHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/watch-providers", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleWatchProviders(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- Get user selections (GET /api/user/watch-providers) ---

func TestGetUserWatchProviders_EmptyByDefault(t *testing.T) {
	h, d, userID := setupWatchProviderHandler(t)
	seedProviders(t, d)

	r := httptest.NewRequest(http.MethodGet, "/api/user/watch-providers", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUserWatchProviders(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var providers []watchProviderDTO
	if err := json.Unmarshal(w.Body.Bytes(), &providers); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}
}

// --- Set user selections (PUT /api/user/watch-providers) ---

func TestSetUserWatchProviders_Success(t *testing.T) {
	h, d, userID := setupWatchProviderHandler(t)
	seedProviders(t, d)

	body := `{"provider_ids":[8,337]}`
	r := httptest.NewRequest(http.MethodPut, "/api/user/watch-providers", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUserWatchProviders(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var providers []watchProviderDTO
	if err := json.Unmarshal(w.Body.Bytes(), &providers); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}

	names := map[string]bool{}
	for _, p := range providers {
		names[p.ProviderName] = true
	}
	if !names["Netflix"] || !names["Disney Plus"] {
		t.Errorf("expected Netflix and Disney Plus, got %v", providers)
	}
}

func TestSetUserWatchProviders_ReplacesExisting(t *testing.T) {
	h, d, userID := setupWatchProviderHandler(t)
	seedProviders(t, d)

	// Set initial selection
	body1 := `{"provider_ids":[8,9]}`
	r1 := httptest.NewRequest(http.MethodPut, "/api/user/watch-providers", bytes.NewBufferString(body1))
	r1 = withUserID(r1, userID)
	w1 := httptest.NewRecorder()
	h.HandleUserWatchProviders(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first PUT: status = %d; body: %s", w1.Code, w1.Body.String())
	}

	// Replace with different selection
	body2 := `{"provider_ids":[337]}`
	r2 := httptest.NewRequest(http.MethodPut, "/api/user/watch-providers", bytes.NewBufferString(body2))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleUserWatchProviders(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
	var providers []watchProviderDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &providers); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	if providers[0].ProviderName != "Disney Plus" {
		t.Errorf("provider = %q, want %q", providers[0].ProviderName, "Disney Plus")
	}
}

func TestSetUserWatchProviders_ClearAll(t *testing.T) {
	h, d, userID := setupWatchProviderHandler(t)
	seedProviders(t, d)

	// Set initial selection
	body1 := `{"provider_ids":[8,9]}`
	r1 := httptest.NewRequest(http.MethodPut, "/api/user/watch-providers", bytes.NewBufferString(body1))
	r1 = withUserID(r1, userID)
	w1 := httptest.NewRecorder()
	h.HandleUserWatchProviders(w1, r1)

	// Clear all
	body2 := `{"provider_ids":[]}`
	r2 := httptest.NewRequest(http.MethodPut, "/api/user/watch-providers", bytes.NewBufferString(body2))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleUserWatchProviders(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
	var providers []watchProviderDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &providers); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}
}

func TestSetUserWatchProviders_InvalidJSON(t *testing.T) {
	h, _, userID := setupWatchProviderHandler(t)

	r := httptest.NewRequest(http.MethodPut, "/api/user/watch-providers", bytes.NewBufferString(`not json`))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUserWatchProviders(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestSetUserWatchProviders_DuplicateIDs(t *testing.T) {
	h, d, userID := setupWatchProviderHandler(t)
	seedProviders(t, d)

	body := `{"provider_ids":[8,8,337]}`
	r := httptest.NewRequest(http.MethodPut, "/api/user/watch-providers", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUserWatchProviders(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var providers []watchProviderDTO
	if err := json.Unmarshal(w.Body.Bytes(), &providers); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(providers) != 2 {
		t.Errorf("expected 2 providers (deduped), got %d", len(providers))
	}
}

func TestSetUserWatchProviders_InvalidProviderIDs(t *testing.T) {
	h, d, userID := setupWatchProviderHandler(t)
	seedProviders(t, d)

	body := `{"provider_ids":[8,99999]}`
	r := httptest.NewRequest(http.MethodPut, "/api/user/watch-providers", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUserWatchProviders(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUserWatchProviders_MethodNotAllowed(t *testing.T) {
	h, _, userID := setupWatchProviderHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/user/watch-providers", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUserWatchProviders(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- User isolation ---

func TestUserWatchProviders_IsolatedPerUser(t *testing.T) {
	d := newTestDB(t)
	h := &WatchProviderHandler{DB: d}
	seedProviders(t, d)

	user1, err := d.CreateUser("Alice", "alice@example.com", "password1")
	if err != nil {
		t.Fatalf("create user1: %v", err)
	}
	user2, err := d.CreateUser("Bob", "bob@example.com", "password1")
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	// User 1 selects Netflix
	body1 := `{"provider_ids":[8]}`
	r1 := httptest.NewRequest(http.MethodPut, "/api/user/watch-providers", bytes.NewBufferString(body1))
	r1 = withUserID(r1, user1.ID)
	w1 := httptest.NewRecorder()
	h.HandleUserWatchProviders(w1, r1)

	// User 2 selects Disney Plus
	body2 := `{"provider_ids":[337]}`
	r2 := httptest.NewRequest(http.MethodPut, "/api/user/watch-providers", bytes.NewBufferString(body2))
	r2 = withUserID(r2, user2.ID)
	w2 := httptest.NewRecorder()
	h.HandleUserWatchProviders(w2, r2)

	// Verify user 1 only sees Netflix
	r3 := httptest.NewRequest(http.MethodGet, "/api/user/watch-providers", nil)
	r3 = withUserID(r3, user1.ID)
	w3 := httptest.NewRecorder()
	h.HandleUserWatchProviders(w3, r3)

	var p1 []watchProviderDTO
	_ = json.Unmarshal(w3.Body.Bytes(), &p1)
	if len(p1) != 1 || p1[0].ProviderName != "Netflix" {
		t.Errorf("user1: expected [Netflix], got %v", p1)
	}

	// Verify user 2 only sees Disney Plus
	r4 := httptest.NewRequest(http.MethodGet, "/api/user/watch-providers", nil)
	r4 = withUserID(r4, user2.ID)
	w4 := httptest.NewRecorder()
	h.HandleUserWatchProviders(w4, r4)

	var p2 []watchProviderDTO
	_ = json.Unmarshal(w4.Body.Bytes(), &p2)
	if len(p2) != 1 || p2[0].ProviderName != "Disney Plus" {
		t.Errorf("user2: expected [Disney Plus], got %v", p2)
	}
}
