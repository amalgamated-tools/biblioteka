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

	"github.com/stretchr/testify/require"
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

	require.Equal(t, http.StatusCreated, w.Code)

	// Verify cache-prevention headers are set.
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Equal(t, "no-cache", w.Header().Get("Pragma"))

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "My Token", resp["name"])
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

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "padded name", capturedName)
}

func TestHandleTokenCreate_EmptyName(t *testing.T) {
	ops := makeTestTokenOps(t)

	body := `{"name":""}`
	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleTokenCreate_WhitespaceOnlyName(t *testing.T) {
	ops := makeTestTokenOps(t)

	body := `{"name":"   "}`
	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleTokenCreate_NameTooLong(t *testing.T) {
	ops := makeTestTokenOps(t)

	longName := strings.Repeat("x", maxTokenNameLength+1)
	body := `{"name":"` + longName + `"}`
	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleTokenCreate_InvalidJSON(t *testing.T) {
	ops := makeTestTokenOps(t)

	r := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader("not-json"))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleTokenCreate(ops, w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
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

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	// Generic error should use the default "failed to create <resource>" message.
	require.Equal(t, "failed to create test token", resp.Error)
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

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	// tokenError.message should be used instead of the generic message.
	require.Equal(t, "failed to generate secure token", resp.Error)
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

	require.Equal(t, http.StatusCreated, w.Code)

	logs, _, err := d.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err, "list audit logs")

	found := false
	for _, l := range logs {
		if l.Action == db.AuditActionAPIKeyCreated && l.EntityID == "entity-abc" {
			found = true
			break
		}
	}
	if !found {
		require.Fail(t, "expected audit log entry was not found")
	}
}

func TestTokenError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("inner cause")
	te := &tokenError{err: inner, message: "client-facing message"}

	require.Equal(t, inner.Error(), te.Error())
	require.ErrorIs(t, te, inner)
}
