package handlers

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
)

// mockEnqueuer records all enqueued jobs for test assertions.
type mockEnqueuer struct {
	mu   sync.Mutex
	jobs []enqueued
	err  error // if set, Enqueue returns this error
}

type enqueued struct {
	Name    string
	Payload json.RawMessage
}

func (m *mockEnqueuer) Enqueue(_ context.Context, name string, payload any) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs = append(m.jobs, enqueued{Name: name, Payload: json.RawMessage(data)})
	return "mock-job-id", nil
}

func setupLibraryHandler(t *testing.T) (*LibraryHandler, string, string) {
	t.Helper()
	d := newTestDB(t)
	h := &LibraryHandler{DB: d}

	admin, err := d.CreateUser(t.Context(), "Admin", "admin@example.com", "password1")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := d.SetAdmin(t.Context(), admin.ID, true); err != nil {
		t.Fatalf("set admin role: %v", err)
	}
	regular, err := d.CreateUser(t.Context(), "Regular", "regular@example.com", "password1")
	if err != nil {
		t.Fatalf("create regular user: %v", err)
	}
	return h, admin.ID, regular.ID
}
