package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// genericMockEnqueuer records enqueued job names and raw payloads.
type genericMockEnqueuer struct {
	mu   sync.Mutex
	jobs []genericEnqueuedJob
	err  error
}

type genericEnqueuedJob struct {
	Name    string
	Payload []byte
}

func (m *genericMockEnqueuer) Enqueue(_ context.Context, name string, payload any) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs = append(m.jobs, genericEnqueuedJob{Name: name, Payload: data})
	return "mock-id", nil
}

func TestScanLibraryHandler(t *testing.T) {
	enq := &genericMockEnqueuer{}
	handler := NewScanLibraryHandler(enq)

	payload, err := json.Marshal(ScanLibraryPayload{
		LibraryID: "lib1",
		Paths:     []string{"/books/fiction", "/books/scifi"},
	})
	require.NoError(t, err, "marshal")
	require.NoError(t, handler(t.Context(), payload), "handler")

	require.Len(t, enq.jobs, 2)
	for i, want := range []string{"/books/fiction", "/books/scifi"} {
		require.Equal(t, JobScanPath, enq.jobs[i].Name, "job[%d] name", i)
		var p ScanPathPayload
		require.NoError(t, json.Unmarshal(enq.jobs[i].Payload, &p), "unmarshal job[%d]", i)
		require.Equal(t, want, p.Path, "job[%d] path", i)
	}
}

func TestScanLibraryHandler_EmptyPaths(t *testing.T) {
	enq := &genericMockEnqueuer{}
	handler := NewScanLibraryHandler(enq)

	payload, err := json.Marshal(ScanLibraryPayload{LibraryID: "lib1"})
	require.NoError(t, err, "marshal")
	require.NoError(t, handler(t.Context(), payload), "handler")

	require.Empty(t, enq.jobs)
}

func TestScanLibraryHandler_EnqueueError(t *testing.T) {
	enq := &genericMockEnqueuer{err: errors.New("redis unavailable")}
	handler := NewScanLibraryHandler(enq)

	payload, err := json.Marshal(ScanLibraryPayload{
		LibraryID: "lib1",
		Paths:     []string{"/books/fiction", "/books/scifi"},
	})
	require.NoError(t, err, "marshal")
	require.NoError(t, handler(t.Context(), payload), "handler should not fail on enqueue errors")

	require.Empty(t, enq.jobs)
}

func TestScanLibraryHandler_MissingLibraryID(t *testing.T) {
	enq := &genericMockEnqueuer{}
	handler := NewScanLibraryHandler(enq)

	payload, err := json.Marshal(ScanLibraryPayload{Paths: []string{"/books/fiction"}})
	require.NoError(t, err, "marshal")
	require.Error(t, handler(t.Context(), payload))
}

func TestScanLibraryHandler_InvalidPayload(t *testing.T) {
	enq := &genericMockEnqueuer{}
	handler := NewScanLibraryHandler(enq)

	require.Error(t, handler(t.Context(), []byte("not json")))
}
