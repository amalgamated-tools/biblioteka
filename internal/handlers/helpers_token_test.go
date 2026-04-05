package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

func Test_HandleTokenCreate(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Token User", "tokens@example.com", "password1")
	require.NoError(t, err, "create user")

	t.Run("empty name yields 400", func(t *testing.T) {
		ops := tokenOps{
			db:              d,
			resource:        "test token",
			auditEntityType: "test_token",
			auditCreate:     db.AuditActionAPIKeyCreated,
			create: func(_ context.Context, _, _ string) (string, any, error) {
				t.Fatal("create should not be called for invalid name")
				return "", nil, nil
			},
		}

		body := mustMarshal(t, map[string]string{"name": ""})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		handleTokenCreate(ops, w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("create error yields 500", func(t *testing.T) {
		d := newTestDB(t)
		user, err := d.CreateUser(t.Context(), "Error User", "error@example.com", "password1")
		require.NoError(t, err, "create user")

		ops := tokenOps{
			db:              d,
			resource:        "test token",
			auditEntityType: "test_token",
			auditCreate:     db.AuditActionAPIKeyCreated,
			create: func(_ context.Context, _, _ string) (string, any, error) {
				return "", nil, errors.New("db failure")
			},
		}

		body := mustMarshal(t, map[string]string{"name": "My Token"})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		handleTokenCreate(ops, w, r)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}

		// Verify no audit log written on error.
		logs, _, err := d.ListAuditLogs(t.Context(), 10, 0)
		require.NoError(t, err, "list audit logs")
		if len(logs) != 0 {
			t.Errorf("expected no audit logs on error, got %d", len(logs))
		}
	})

	t.Run("success returns 201 with no-store headers", func(t *testing.T) {
		type tokenResp struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		}
		ops := tokenOps{
			db:              d,
			resource:        "test token",
			auditEntityType: "test_token",
			auditCreate:     db.AuditActionAPIKeyCreated,
			create: func(_ context.Context, _, _ string) (string, any, error) {
				return "entity-123", tokenResp{ID: "entity-123", Token: "secret"}, nil
			},
		}

		body := mustMarshal(t, map[string]string{"name": "My Token"})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		handleTokenCreate(ops, w, r)

		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
		if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("Cache-Control = %q, want %q", cc, "no-store")
		}
		var resp tokenResp
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
		if resp.Token != "secret" {
			t.Errorf("token = %q, want %q", resp.Token, "secret")
		}
	})

	t.Run("success writes audit log", func(t *testing.T) {
		ops := tokenOps{
			db:              d,
			resource:        "audit token",
			auditEntityType: "audit_token",
			auditCreate:     db.AuditActionAPIKeyCreated,
			create: func(_ context.Context, _, _ string) (string, any, error) {
				return "audit-entity-1", map[string]string{"token": "val"}, nil
			},
		}

		body := mustMarshal(t, map[string]string{"name": "Audited"})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		handleTokenCreate(ops, w, r)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
		}

		logs, _, err := d.ListAuditLogs(t.Context(), 10, 0)
		require.NoError(t, err, "list audit logs")
		found := false
		for _, l := range logs {
			if l.Action == db.AuditActionAPIKeyCreated && l.EntityID == "audit-entity-1" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected audit log entry for created token")
		}
	})

	t.Run("tokenError returns specific message", func(t *testing.T) {
		d := newTestDB(t)
		user, err := d.CreateUser(t.Context(), "Test User", "test3@example.com", "password1")
		require.NoError(t, err, "create user")

		ops := tokenOps{
			db:              d,
			resource:        "widget",
			auditEntityType: "widget",
			auditCreate:     db.AuditActionAPIKeyCreated,
			create: func(_ context.Context, _, _ string) (string, any, error) {
				return "", nil, &tokenError{err: errors.New("rng failure"), message: "failed to generate widget"}
			},
		}

		body := mustMarshal(t, map[string]string{"name": "My Widget"})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		handleTokenCreate(ops, w, r)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
		var result map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "unmarshal")
		if result["error"] != "failed to generate widget" {
			t.Errorf("error = %q, want %q", result["error"], "failed to generate widget")
		}
	})
}
