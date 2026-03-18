package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
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

	payload, _ := json.Marshal(ScanLibraryPayload{
		LibraryID: "lib1",
		Paths:     []string{"/books/fiction", "/books/scifi"},
	})
	if err := handler(context.Background(), payload); err != nil {
		failNowf(t, "handler: %v", err)
	}

	if got := len(enq.jobs); got != 2 {
		failNowf(t, "expected 2 enqueued jobs, got %d", got)
	}
	for i, want := range []string{"/books/fiction", "/books/scifi"} {
		if enq.jobs[i].Name != JobScanPath {
			failf(t, "job[%d] name = %q, want %q", i, enq.jobs[i].Name, JobScanPath)
		}
		var p ScanPathPayload
		if err := json.Unmarshal(enq.jobs[i].Payload, &p); err != nil {
			failNowf(t, "unmarshal job[%d]: %v", i, err)
		}
		if p.Path != want {
			failf(t, "job[%d] path = %q, want %q", i, p.Path, want)
		}
	}
}

func TestScanLibraryHandler_EmptyPaths(t *testing.T) {
	enq := &genericMockEnqueuer{}
	handler := NewScanLibraryHandler(enq)

	payload, _ := json.Marshal(ScanLibraryPayload{LibraryID: "lib1"})
	if err := handler(context.Background(), payload); err != nil {
		failNowf(t, "handler: %v", err)
	}

	if len(enq.jobs) != 0 {
		failf(t, "expected 0 enqueued jobs, got %d", len(enq.jobs))
	}
}

func TestScanLibraryHandler_EnqueueError(t *testing.T) {
	enq := &genericMockEnqueuer{err: errors.New("redis unavailable")}
	handler := NewScanLibraryHandler(enq)

	payload, _ := json.Marshal(ScanLibraryPayload{
		LibraryID: "lib1",
		Paths:     []string{"/books/fiction", "/books/scifi"},
	})
	if err := handler(context.Background(), payload); err != nil {
		failNowf(t, "handler should not fail on enqueue errors: %v", err)
	}

	if len(enq.jobs) != 0 {
		failf(t, "expected 0 enqueued jobs, got %d", len(enq.jobs))
	}
}

func TestScanLibraryHandler_MissingLibraryID(t *testing.T) {
	enq := &genericMockEnqueuer{}
	handler := NewScanLibraryHandler(enq)

	payload, _ := json.Marshal(ScanLibraryPayload{Paths: []string{"/books/fiction"}})
	if err := handler(context.Background(), payload); err == nil {
		failNow(t, "expected error when library_id is missing")
	}
}

func TestScanLibraryHandler_InvalidPayload(t *testing.T) {
	enq := &genericMockEnqueuer{}
	handler := NewScanLibraryHandler(enq)

	if err := handler(context.Background(), []byte("not json")); err == nil {
		failNow(t, "expected error for invalid payload")
	}
}
