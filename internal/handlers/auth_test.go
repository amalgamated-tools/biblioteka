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

	"github.com/stretchr/testify/require"
)

// newTestDB creates an in-memory SQLite database with all real migrations applied.
func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err, "newTestDB: open")
	t.Cleanup(func() { _ = sqlDB.Close() })

	err = db.RunMigrations(t.Context(), sqlDB, db.DialectSQLite)
	require.NoError(t, err, "newTestDB: migrations")

	return &db.DB{DB: sqlDB, Dialect: db.DialectSQLite}
}

func newTestJWT(t *testing.T) *auth.JWTManager {
	t.Helper()
	jm, err := auth.NewJWTManager("testsecret", time.Hour)
	require.NoError(t, err, "newTestJWT")
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

	require.Equal(t, http.StatusCreated, w.Code)
	var resp authResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.NotEqual(t, "", resp.Token)
	require.Equal(t, "alice@example.com", resp.User.Email)
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
			require.Equal(t, http.StatusBadRequest, w.Code)
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

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignup_DuplicateEmail(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	body := `{"name":"Alice","email":"alice@example.com","password":"secret123"}`
	// First signup succeeds
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Signup(w, r)
	require.Equal(t, http.StatusCreated, w.Code)

	// Second signup with same email should fail
	r2 := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	w2 := httptest.NewRecorder()
	h.Signup(w2, r2)
	require.Equal(t, http.StatusConflict, w2.Code)
}

func TestSignup_InvalidBody(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	h.Signup(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignup_MethodNotAllowed(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodGet, "/api/auth/signup", nil)
	w := httptest.NewRecorder()
	h.Signup(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestLogin_Success(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	signupBody := `{"name":"Bob","email":"bob@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(signupBody))
	w := httptest.NewRecorder()
	h.Signup(w, r)
	require.Equal(t, http.StatusCreated, w.Code)

	loginBody := `{"email":"bob@example.com","password":"secret123"}`
	r2 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	w2 := httptest.NewRecorder()
	h.Login(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code)
	var resp authResponse
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp), "unmarshal")
	require.NotEqual(t, "", resp.Token)
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

	require.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestLogin_UserNotFound(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	body := `{"email":"nobody@example.com","password":"password1"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, r)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_MissingFields(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	body := `{"email":"","password":""}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_MethodNotAllowed(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	w := httptest.NewRecorder()
	h.Login(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
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

	require.Equal(t, http.StatusOK, w2.Code)
	var resp userDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "dave@example.com", resp.Email)
}

func TestMe_UserNotFound(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r = withUserID(r, "nonexistent-user-id")
	w := httptest.NewRecorder()
	h.Me(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestMe_MethodNotAllowed(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodPost, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	h.Me(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
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

	require.Equal(t, http.StatusOK, w2.Code)

	// Verify new password works at login
	loginBody := `{"email":"eve@example.com","password":"newpassword1"}`
	r3 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	w3 := httptest.NewRecorder()
	h.Login(w3, r3)
	require.Equal(t, http.StatusOK, w3.Code)
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

	require.Equal(t, http.StatusUnauthorized, w2.Code)
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

	require.Equal(t, http.StatusBadRequest, w2.Code)
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

	require.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestChangePassword_MethodNotAllowed(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodGet, "/api/auth/password", nil)
	w := httptest.NewRecorder()
	h.ChangePassword(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
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
	require.NotNil(t, found)
	require.NotEmpty(t, found.Value)
	require.Equal(t, true, found.HttpOnly)
	require.Equal(t, http.SameSiteStrictMode, found.SameSite)
	require.Equal(t, "/", found.Path)
}

func TestSignup_SetsCookie(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	body := `{"name":"CookieUser","email":"cookie@example.com","password":"secret123"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Signup(w, r)

	require.Equal(t, http.StatusCreated, w.Code)
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
	require.Equal(t, http.StatusCreated, w.Code)

	loginBody := `{"email":"cookielogin@example.com","password":"secret123"}`
	r2 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	w2 := httptest.NewRecorder()
	h.Login(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code)
	assertAuthCookie(t, w2, true)
}

func TestLogout_ClearsCookie(t *testing.T) {
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

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
	d := newTestDB(t)
	h := &AuthHandler{DB: d, JWT: newTestJWT(t)}

	r := httptest.NewRequest(http.MethodGet, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	h.Logout(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
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
	require.Equal(t, http.StatusNotFound, w.Code)
}
