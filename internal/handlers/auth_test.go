package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

func newAuthHandler(t *testing.T) *AuthHandler {
	t.Helper()
	return &AuthHandler{DB: newTestDB(t), JWT: newTestJWT(t)}
}

// mustSignup creates a new user via the signup endpoint and returns the auth response.
// It fails the test if signup does not return 201 Created.
func mustSignup(t *testing.T, h *AuthHandler, name, email, password string) authResponse {
	t.Helper()
	body, err := json.Marshal(map[string]string{"name": name, "email": email, "password": password})
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Signup(w, r)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp authResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func TestSignup_Success(t *testing.T) {
	h := newAuthHandler(t)

	body := `{"name":"Alice","email":"alice@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Signup(w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp authResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.NotEqual(t, "", resp.Token)
	require.Equal(t, "alice@example.com", resp.User.Email)
}

func TestSignup_MissingFields(t *testing.T) {
	h := newAuthHandler(t)

	tests := []struct {
		name string
		body string
	}{
		{"missing name", `{"email":"a@b.com","password":"secret123"}`},
		{"missing email", `{"name":"Alice","password":"secret123"}`},
		{"missing password", `{"name":"Alice","email":"a@b.com"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()
			h.Signup(w, r)
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestSignup_ShortPassword(t *testing.T) {
	h := newAuthHandler(t)

	body := `{"name":"Alice","email":"alice@example.com","password":"123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Signup(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignup_DuplicateEmail(t *testing.T) {
	h := newAuthHandler(t)

	body := `{"name":"Alice","email":"alice@example.com","password":"secret123"}`
	// First signup succeeds.
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Signup(w, r)
	require.Equal(t, http.StatusCreated, w.Code)

	// Second signup with same email should fail.
	r2 := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	w2 := httptest.NewRecorder()
	h.Signup(w2, r2)
	require.Equal(t, http.StatusConflict, w2.Code)
}

func TestSignup_InvalidBody(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	h.Signup(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignup_MethodNotAllowed(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/signup", nil)
	w := httptest.NewRecorder()
	h.Signup(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestSignup_Disabled(t *testing.T) {
	h := newAuthHandler(t)
	h.DisableSignup = true

	body := `{"name":"Alice","email":"alice@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Signup(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "signup is disabled", resp["error"])
}

func TestLogin_Success(t *testing.T) {
	h := newAuthHandler(t)
	mustSignup(t, h, "Bob", "bob@example.com", "secret123")

	loginBody := `{"email":"bob@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	w := httptest.NewRecorder()
	h.Login(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp authResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.NotEqual(t, "", resp.Token)
}

func TestLogin_WrongPassword(t *testing.T) {
	h := newAuthHandler(t)
	mustSignup(t, h, "Carol", "carol@example.com", "correctpw")

	loginBody := `{"email":"carol@example.com","password":"wrongpw12"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	w := httptest.NewRecorder()
	h.Login(w, r)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_UserNotFound(t *testing.T) {
	h := newAuthHandler(t)

	body := `{"email":"nobody@example.com","password":"password1"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, r)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "invalid email or password", resp.Error)
}

// Regression test for account enumeration via OIDC login error (PR #1713).
// An OIDC-only account (empty password_hash) must return the same generic
// "invalid email or password" error as a non-existent account so that
// attackers cannot use the login endpoint to discover which emails are
// registered as OIDC-only vs. not registered at all.
func TestLogin_OIDCOnlyAccountReturnsGenericError(t *testing.T) {
	h := newAuthHandler(t)

	_, err := h.DB.CreateOIDCUser(t.Context(), "Alice", "alice@example.com", "oidc-subject-123")
	require.NoError(t, err)

	body := `{"email":"alice@example.com","password":"somepassword"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, r)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// Must match the user-not-found error exactly — no "OIDC" or account-type hint.
	require.Equal(t, "invalid email or password", resp.Error)
}

func TestLogin_MissingFields(t *testing.T) {
	h := newAuthHandler(t)

	body := `{"email":"","password":""}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_MethodNotAllowed(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	w := httptest.NewRecorder()
	h.Login(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestMe_Success(t *testing.T) {
	h := newAuthHandler(t)
	signupResp := mustSignup(t, h, "Dave", "dave@example.com", "secret123")

	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r = withUserID(r, signupResp.User.ID)
	w := httptest.NewRecorder()
	h.Me(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp userDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "dave@example.com", resp.Email)
	require.Equal(t, "Dave", resp.Name)
}

func TestMe_UserNotFound(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r = withUserID(r, "nonexistent-user-id")
	w := httptest.NewRecorder()
	h.Me(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestMe_EmptyUserID(t *testing.T) {
	h := newAuthHandler(t)

	// Inject an empty user ID via the proper context key; lookup will fail.
	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r = withUserID(r, "")
	w := httptest.NewRecorder()
	h.Me(w, r)

	// GetUserByID("") returns sql.ErrNoRows which the handler maps to 404.
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestMe_MethodNotAllowed(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	h.Me(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestChangePassword_Success(t *testing.T) {
	h := newAuthHandler(t)
	signupResp := mustSignup(t, h, "Eve", "eve@example.com", "oldpassword")

	// Change password.
	changeBody := `{"currentPassword":"oldpassword","newPassword":"newpassword1"}`
	r := httptest.NewRequest(http.MethodPut, "/api/auth/password", bytes.NewBufferString(changeBody))
	r = withUserID(r, signupResp.User.ID)
	w := httptest.NewRecorder()
	h.ChangePassword(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	// Verify audit log was created
	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err)
	found := false
	for _, l := range logs {
		if l.Action == db.AuditActionPasswordChanged && l.EntityID == signupResp.User.ID {
			found = true
			break
		}
	}
	require.True(t, found, "expected audit log entry for password change")

	// Verify new password works at login.
	loginBody := `{"email":"eve@example.com","password":"newpassword1"}`
	r2 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	w2 := httptest.NewRecorder()
	h.Login(w2, r2)
	require.Equal(t, http.StatusOK, w2.Code)
}

func TestChangePassword_Invalid(t *testing.T) {
	h := newAuthHandler(t)
	signupResp := mustSignup(t, h, "Frank", "frank@example.com", "correctpw")

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"wrong current password", `{"currentPassword":"wrongpassword","newPassword":"newpassword1"}`, http.StatusUnauthorized},
		{"short new password", `{"currentPassword":"correctpw","newPassword":"abc"}`, http.StatusBadRequest},
		{"missing fields", `{"currentPassword":"","newPassword":""}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPut, "/api/auth/password", bytes.NewBufferString(tt.body))
			r = withUserID(r, signupResp.User.ID)
			w := httptest.NewRecorder()
			h.ChangePassword(w, r)
			require.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestChangePassword_MethodNotAllowed(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/password", nil)
	w := httptest.NewRecorder()
	h.ChangePassword(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// TestChangePassword_OIDCOnlyAccount verifies that an OIDC-only account
// (empty PasswordHash) cannot change its password via the local credential flow.
// This is a security boundary: OIDC users have no local password to verify
// against, so the endpoint must reject the request regardless of the supplied
// currentPassword value.
func TestChangePassword_OIDCOnlyAccount(t *testing.T) {
	h := newAuthHandler(t)
	user, err := h.DB.CreateOIDCUser(t.Context(), "OIDCUser", "oidconly@example.com", "oidc-subject-abc")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPut, "/api/auth/password",
		bytes.NewBufferString(`{"currentPassword":"anypassword","newPassword":"newpassword1"}`))
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()
	h.ChangePassword(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Contains(t, resp.Error, "cannot change password")
}

// --- Set-Cookie header assertions ---

// assertAuthCookie checks that the response contains a Set-Cookie header for
// the auth cookie with the expected attributes.
func assertAuthCookie(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.TokenCookieName() {
			found = c
			break
		}
	}
	require.NotNil(t, found)
	require.NotEmpty(t, found.Value)
	require.True(t, found.HttpOnly)
	require.Equal(t, http.SameSiteStrictMode, found.SameSite)
	require.Equal(t, "/", found.Path)
}

func TestSignup_SetsCookie(t *testing.T) {
	h := newAuthHandler(t)

	body := `{"name":"CookieUser","email":"cookie@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Signup(w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	assertAuthCookie(t, w)
}

func TestLogin_SetsCookie(t *testing.T) {
	h := newAuthHandler(t)
	mustSignup(t, h, "CookieLogin", "cookielogin@example.com", "secret123")

	loginBody := `{"email":"cookielogin@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	w := httptest.NewRecorder()
	h.Login(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assertAuthCookie(t, w)
}

func TestLogout_ClearsCookie(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	r.Header.Set("Origin", "http://"+r.Host)
	w := httptest.NewRecorder()
	h.Logout(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.TokenCookieName() {
			found = c
			break
		}
	}
	require.NotNil(t, found)
	require.Equal(t, -1, found.MaxAge)
	require.Equal(t, "", found.Value)
}

func TestLogout_MethodNotAllowed(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	h.Logout(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestUpdateProfile_Success(t *testing.T) {
	h := newAuthHandler(t)
	signupResp := mustSignup(t, h, "Original Name", "profile@example.com", "secret123")
	require.Equal(t, "Original Name", signupResp.User.Name)

	// Update the display name.
	updateBody := `{"name":"Updated Name"}`
	r := httptest.NewRequest(http.MethodPut, "/api/auth/me", bytes.NewBufferString(updateBody))
	r = withUserID(r, signupResp.User.ID)
	w := httptest.NewRecorder()
	h.Me(w, r)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	var resp userDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "Updated Name", resp.Name)
	require.Equal(t, "profile@example.com", resp.Email)

	// Verify the name persisted via GET /api/auth/me.
	r2 := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r2 = withUserID(r2, signupResp.User.ID)
	w2 := httptest.NewRecorder()
	h.Me(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code)
	var getResp userDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &getResp))
	require.Equal(t, "Updated Name", getResp.Name)
}

func TestUpdateProfile_EmptyName(t *testing.T) {
	h := newAuthHandler(t)
	signupResp := mustSignup(t, h, "Alice", "alice2@example.com", "secret123")

	tests := []struct {
		name string
		body string
	}{
		{"empty string", `{"name":""}`},
		{"whitespace only", `{"name":"   "}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPut, "/api/auth/me", bytes.NewBufferString(tt.body))
			r = withUserID(r, signupResp.User.ID)
			w := httptest.NewRecorder()
			h.Me(w, r)
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestUpdateProfile_MethodNotAllowed(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	h.Me(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	h := newAuthHandler(t)

	body := `{"name":"New Name"}`
	r := httptest.NewRequest(http.MethodPut, "/api/auth/me", bytes.NewBufferString(body))
	r = withUserID(r, "nonexistent-id")
	w := httptest.NewRecorder()
	h.Me(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}
