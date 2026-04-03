package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

func makeTestTokenOps(t *testing.T) tokenOps {
	t.Helper()
	d := newTestDB(t)
	return tokenOps{
		db:              d,
		resource:        "test token",
		auditEntityType: "test_token",
		auditCreate:     db.AuditActionAPIKeyCreated,
		create: func(_ context.Context, _, name string) (string, any, error) {
			return "entity-1", map[string]string{"name": name, "token": "raw-token-value"}, nil
		},
	}
}

func TestHandleTokenCreate_Success(t *testing.T) {
	ops := makeTestTokenOps(t)

	body := `{"name":"My Token"}`
	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Verify cache-prevention headers are set.
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-store")
	}
	if pragma := w.Header().Get("Pragma"); pragma != "no-cache" {
		t.Errorf("Pragma = %q, want %q", pragma, "no-cache")
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["name"] != "My Token" {
		t.Errorf("name = %q, want %q", resp["name"], "My Token")
	}
}

func TestHandleTokenCreate_NameTrimmed(t *testing.T) {
	ops := makeTestTokenOps(t)
	var capturedName string
	ops.create = func(_ context.Context, _, name string) (string, any, error) {
		capturedName = name
		return "entity-1", map[string]string{"name": name}, nil
	}

	body := `{"name":"  padded name  "}`
	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if capturedName != "padded name" {
		t.Errorf("create received name %q, want trimmed %q", capturedName, "padded name")
	}
}

func TestHandleTokenCreate_EmptyName(t *testing.T) {
	ops := makeTestTokenOps(t)

	body := `{"name":""}`
	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleTokenCreate_WhitespaceOnlyName(t *testing.T) {
	ops := makeTestTokenOps(t)

	body := `{"name":"   "}`
	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleTokenCreate_NameTooLong(t *testing.T) {
	ops := makeTestTokenOps(t)

	longName := strings.Repeat("x", maxTokenNameLength+1)
	body := `{"name":"` + longName + `"}`
	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleTokenCreate_InvalidJSON(t *testing.T) {
	ops := makeTestTokenOps(t)

	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader("not-json"))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleTokenCreate_GenericCreateError(t *testing.T) {
	ops := makeTestTokenOps(t)
	ops.create = func(_ context.Context, _, _ string) (string, any, error) {
		return "", nil, errors.New("db write failed")
	}

	body := `{"name":"My Token"}`
	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Generic error should use the default "failed to create <resource>" message.
	if resp.Error != "failed to create test token" {
		t.Errorf("error = %q, want %q", resp.Error, "failed to create test token")
	}
}

func TestHandleTokenCreate_TokenErrorMessage(t *testing.T) {
	ops := makeTestTokenOps(t)
	ops.create = func(_ context.Context, _, _ string) (string, any, error) {
		return "", nil, &tokenError{
			err:     errors.New("internal cause"),
			message: "failed to generate secure token",
		}
	}

	body := `{"name":"My Token"}`
	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// tokenError.message should be used instead of the generic message.
	if resp.Error != "failed to generate secure token" {
		t.Errorf("error = %q, want %q", resp.Error, "failed to generate secure token")
	}
}

func TestHandleTokenCreate_AuditLog(t *testing.T) {
	d := newTestDB(t)
	ops := tokenOps{
		db:              d,
		resource:        "test token",
		auditEntityType: "test_token",
		auditCreate:     db.AuditActionAPIKeyCreated,
		create: func(_ context.Context, _, name string) (string, any, error) {
			return "entity-abc", map[string]string{"name": name}, nil
		},
	}

	body := `{"name":"Audited Token"}`
	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(body))
	r = withUserID(r, "user-audit")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	logs, _, err := d.ListAuditLogs(t.Context(), 10, 0)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}

	found := false
	for _, l := range logs {
		if l.Action == db.AuditActionAPIKeyCreated && l.EntityID == "entity-abc" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit log entry was not found")
	}
}

func TestTokenError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("inner cause")
	te := &tokenError{err: inner, message: "client-facing message"}

	if te.Error() != inner.Error() {
		t.Errorf("Error() = %q, want %q", te.Error(), inner.Error())
	}
	if !errors.Is(te, inner) {
		t.Error("errors.Is(te, inner) = false, want true")
	}
}
