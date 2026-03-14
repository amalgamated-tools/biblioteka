package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	_ "modernc.org/sqlite"
)

// newTestDB creates an in-memory SQLite database with all real migrations applied.
func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("newTestDB: open: %v", err)
	}
	if err := db.RunMigrations(sqlDB, db.DialectSQLite); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("newTestDB: migrations: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return &db.DB{DB: sqlDB, Dialect: db.DialectSQLite}
}

func newTestJWT(t *testing.T) *auth.JWTManager {
	t.Helper()
	jm, err := auth.NewJWTManager("testsecret", time.Hour)
	if err != nil {
		t.Fatalf("newTestJWT: %v", err)
	}
	return jm
}

// withUserID returns a copy of r with the user ID injected into the context.
func withUserID(r *http.Request, userID string) *http.Request {
	ctx := auth.ContextWithUserID(r.Context(), userID)
	return r.WithContext(ctx)
}

func TestSignup_Success(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	body := `{"name":"Alice","email":"alice@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Signup(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	var resp authResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.User.Email != "alice@example.com" {
		t.Errorf("email = %q, want %q", resp.User.Email, "alice@example.com")
	}
}

func TestSignup_MissingFields(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

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
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestSignup_ShortPassword(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	body := `{"name":"Alice","email":"alice@example.com","password":"123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Signup(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	body := `{"name":"Alice","email":"alice@example.com","password":"secret123"}`
	// First signup succeeds
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Signup(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("first signup failed: %s", w.Body.String())
	}

	// Second signup with same email should fail
	r2 := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	w2 := httptest.NewRecorder()
	h.Signup(w2, r2)
	if w2.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusConflict)
	}
}

func TestSignup_InvalidBody(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	h.Signup(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSignup_MethodNotAllowed(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodGet, "/api/auth/signup", nil)
	w := httptest.NewRecorder()
	h.Signup(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestLogin_Success(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	signupBody := `{"name":"Bob","email":"bob@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(signupBody))
	w := httptest.NewRecorder()
	h.Signup(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("signup failed: %s", w.Body.String())
	}

	loginBody := `{"email":"bob@example.com","password":"secret123"}`
	r2 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	w2 := httptest.NewRecorder()
	h.Login(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
	var resp authResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	signupBody := `{"name":"Carol","email":"carol@example.com","password":"correctpw"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(signupBody))
	w := httptest.NewRecorder()
	h.Signup(w, r)

	loginBody := `{"email":"carol@example.com","password":"wrongpw12"}`
	r2 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	w2 := httptest.NewRecorder()
	h.Login(w2, r2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusUnauthorized)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	body := `{"email":"nobody@example.com","password":"password1"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogin_MissingFields(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	body := `{"email":"","password":""}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLogin_MethodNotAllowed(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestMe_Success(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	// Sign up to create a user
	signupBody := `{"name":"Dave","email":"dave@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(signupBody))
	w := httptest.NewRecorder()
	h.Signup(w, r)
	var signupResp authResponse
	_ = json.Unmarshal(w.Body.Bytes(), &signupResp)

	r2 := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r2 = withUserID(r2, signupResp.User.ID)
	w2 := httptest.NewRecorder()
	h.Me(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
	var resp userDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Email != "dave@example.com" {
		t.Errorf("email = %q, want %q", resp.Email, "dave@example.com")
	}
}

func TestMe_UserNotFound(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r = withUserID(r, "nonexistent-user-id")
	w := httptest.NewRecorder()
	h.Me(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMe_MethodNotAllowed(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodPost, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	h.Me(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestChangePassword_Success(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	// Sign up
	signupBody := `{"name":"Eve","email":"eve@example.com","password":"oldpassword"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(signupBody))
	w := httptest.NewRecorder()
	h.Signup(w, r)
	var signupResp authResponse
	_ = json.Unmarshal(w.Body.Bytes(), &signupResp)

	// Change password
	changeBody := `{"currentPassword":"oldpassword","newPassword":"newpassword1"}`
	r2 := httptest.NewRequest(http.MethodPut, "/api/auth/password", bytes.NewBufferString(changeBody))
	r2 = withUserID(r2, signupResp.User.ID)
	w2 := httptest.NewRecorder()
	h.ChangePassword(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	// Verify new password works at login
	loginBody := `{"email":"eve@example.com","password":"newpassword1"}`
	r3 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	w3 := httptest.NewRecorder()
	h.Login(w3, r3)
	if w3.Code != http.StatusOK {
		t.Errorf("login with new password: status = %d, want %d", w3.Code, http.StatusOK)
	}
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	signupBody := `{"name":"Frank","email":"frank@example.com","password":"correctpw"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(signupBody))
	w := httptest.NewRecorder()
	h.Signup(w, r)
	var signupResp authResponse
	_ = json.Unmarshal(w.Body.Bytes(), &signupResp)

	changeBody := `{"currentPassword":"wrongpassword","newPassword":"newpassword1"}`
	r2 := httptest.NewRequest(http.MethodPut, "/api/auth/password", bytes.NewBufferString(changeBody))
	r2 = withUserID(r2, signupResp.User.ID)
	w2 := httptest.NewRecorder()
	h.ChangePassword(w2, r2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusUnauthorized)
	}
}

func TestChangePassword_ShortNewPassword(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	signupBody := `{"name":"Grace","email":"grace@example.com","password":"correctpw"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(signupBody))
	w := httptest.NewRecorder()
	h.Signup(w, r)
	var signupResp authResponse
	_ = json.Unmarshal(w.Body.Bytes(), &signupResp)

	changeBody := `{"currentPassword":"correctpw","newPassword":"abc"}`
	r2 := httptest.NewRequest(http.MethodPut, "/api/auth/password", bytes.NewBufferString(changeBody))
	r2 = withUserID(r2, signupResp.User.ID)
	w2 := httptest.NewRecorder()
	h.ChangePassword(w2, r2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusBadRequest)
	}
}

func TestChangePassword_MissingFields(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	signupBody := `{"name":"Hank","email":"hank@example.com","password":"correctpw"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(signupBody))
	w := httptest.NewRecorder()
	h.Signup(w, r)
	var signupResp authResponse
	_ = json.Unmarshal(w.Body.Bytes(), &signupResp)

	changeBody := `{"currentPassword":"","newPassword":""}`
	r2 := httptest.NewRequest(http.MethodPut, "/api/auth/password", bytes.NewBufferString(changeBody))
	r2 = withUserID(r2, signupResp.User.ID)
	w2 := httptest.NewRecorder()
	h.ChangePassword(w2, r2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusBadRequest)
	}
}

func TestChangePassword_MethodNotAllowed(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodGet, "/api/auth/password", nil)
	w := httptest.NewRecorder()
	h.ChangePassword(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- Set-Cookie header assertions ---

// assertAuthCookie checks that the response contains a Set-Cookie header for
// the auth cookie with the expected attributes.
func assertAuthCookie(t *testing.T, w *httptest.ResponseRecorder, wantValue bool) {
	t.Helper()
	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.TokenCookieName() {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected Set-Cookie header for auth cookie, but none found")
	}
	if wantValue && found.Value == "" {
		t.Error("expected non-empty cookie value")
	}
	if found.HttpOnly != true {
		t.Error("expected HttpOnly to be true")
	}
	if found.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, want StrictMode", found.SameSite)
	}
	if found.Path != "/" {
		t.Errorf("Path = %q, want %q", found.Path, "/")
	}
}

func TestSignup_SetsCookie(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	body := `{"name":"CookieUser","email":"cookie@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Signup(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	assertAuthCookie(t, w, true)
}

func TestLogin_SetsCookie(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	// Create user first
	signupBody := `{"name":"CookieLogin","email":"cookielogin@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(signupBody))
	w := httptest.NewRecorder()
	h.Signup(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("signup failed: %s", w.Body.String())
	}

	loginBody := `{"email":"cookielogin@example.com","password":"secret123"}`
	r2 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	w2 := httptest.NewRecorder()
	h.Login(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
	assertAuthCookie(t, w2, true)
}

func TestLogout_ClearsCookie(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	h.Logout(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.TokenCookieName() {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected Set-Cookie header for auth cookie on logout, but none found")
	}
	if found.MaxAge != -1 {
		t.Errorf("MaxAge = %d, want -1 (cookie deletion)", found.MaxAge)
	}
	if found.Value != "" {
		t.Errorf("cookie value = %q, want empty", found.Value)
	}
}

func TestLogout_MethodNotAllowed(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodGet, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	h.Logout(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestMe_EmptyUserID(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	// Inject an empty user ID via the proper context key; lookup will fail.
	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r = withUserID(r, "")
	w := httptest.NewRecorder()
	h.Me(w, r)

	// GetUserByID("") returns sql.ErrNoRows which the handler maps to 404.
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
