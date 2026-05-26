package handlers

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/stretchr/testify/require"
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

func (m *mockEnqueuer) Enqueue(_ context.Context, name string, payload any, _ ...jobs.EnqueueOption) (string, error) {
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
	require.NoError(t, err, "create admin")
	require.NoError(t, d.SetAdmin(t.Context(), admin.ID, true), "set admin role")
	regular, err := d.CreateUser(t.Context(), "Regular", "regular@example.com", "password1")
	require.NoError(t, err, "create regular user")
	return h, admin.ID, regular.ID
}
