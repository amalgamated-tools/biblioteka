package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// mockLibraryLister records calls to ListLibraries and returns a preset list.
type mockLibraryLister struct {
	libraries []db.Library
	err       error
}

func (m *mockLibraryLister) ListLibraries() ([]db.Library, error) {
	return m.libraries, m.err
}

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

func TestScanLibrary(t *testing.T) {
	enq := &genericMockEnqueuer{}
	n := ScanLibrary(context.Background(), enq, "lib1", []string{"/books/fiction", "/books/scifi"})

	if n != 2 {
		t.Errorf("ScanLibrary returned %d, want 2", n)
	}
	if got := len(enq.jobs); got != 2 {
		t.Fatalf("expected 2 enqueued jobs, got %d", got)
	}
	for i, want := range []string{"/books/fiction", "/books/scifi"} {
		if enq.jobs[i].Name != JobScanPath {
			t.Errorf("job[%d] name = %q, want %q", i, enq.jobs[i].Name, JobScanPath)
		}
		var p ScanPathPayload
		if err := json.Unmarshal(enq.jobs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal job[%d]: %v", i, err)
		}
		if p.Path != want {
			t.Errorf("job[%d] path = %q, want %q", i, p.Path, want)
		}
	}
}

func TestScanLibrary_EmptyPaths(t *testing.T) {
	enq := &genericMockEnqueuer{}
	n := ScanLibrary(context.Background(), enq, "lib1", nil)

	if n != 0 {
		t.Errorf("ScanLibrary returned %d, want 0", n)
	}
	if len(enq.jobs) != 0 {
		t.Errorf("expected 0 enqueued jobs, got %d", len(enq.jobs))
	}
}

func TestScanLibrary_EnqueueError(t *testing.T) {
	enq := &genericMockEnqueuer{err: errors.New("redis unavailable")}
	n := ScanLibrary(context.Background(), enq, "lib1", []string{"/books/fiction", "/books/scifi"})

	if n != 0 {
		t.Errorf("ScanLibrary returned %d, want 0", n)
	}
}

func TestScanLibrariesHandler(t *testing.T) {
	enq := &genericMockEnqueuer{}
	lister := &mockLibraryLister{
		libraries: []db.Library{
			{
				ID:        "lib1",
				Name:      "Fiction",
				Paths:     `["/books/fiction","/books/scifi"]`,
				Monitored: true,
			},
			{
				ID:        "lib2",
				Name:      "Non-Fiction",
				Paths:     `["/books/nonfiction"]`,
				Monitored: true,
			},
			{
				ID:        "lib3",
				Name:      "Archive",
				Paths:     `["/books/archive"]`,
				Monitored: false, // should be skipped
			},
		},
	}

	handler := NewScanLibrariesHandler(lister, enq)
	if err := handler(context.Background(), nil); err != nil {
		t.Fatalf("handler: %v", err)
	}

	// Expect 3 scan:path jobs (2 from lib1, 1 from lib2; lib3 is not monitored)
	if got := len(enq.jobs); got != 3 {
		t.Errorf("expected 3 enqueued jobs, got %d", got)
	}

	for _, j := range enq.jobs {
		if j.Name != JobScanPath {
			t.Errorf("expected job name %q, got %q", JobScanPath, j.Name)
		}
		var p ScanPathPayload
		if err := json.Unmarshal(j.Payload, &p); err != nil {
			t.Errorf("unmarshal payload: %v", err)
		}
		if p.Path == "" {
			t.Error("enqueued job has empty path")
		}
	}
}

func TestScanLibrariesHandler_NoMonitoredLibraries(t *testing.T) {
	enq := &genericMockEnqueuer{}
	lister := &mockLibraryLister{
		libraries: []db.Library{
			{ID: "lib1", Name: "Archive", Paths: `["/books/archive"]`, Monitored: false},
		},
	}

	handler := NewScanLibrariesHandler(lister, enq)
	if err := handler(context.Background(), nil); err != nil {
		t.Fatalf("handler: %v", err)
	}

	if len(enq.jobs) != 0 {
		t.Errorf("expected 0 enqueued jobs, got %d", len(enq.jobs))
	}
}

func TestScanLibrariesHandler_EmptyLibraryList(t *testing.T) {
	enq := &genericMockEnqueuer{}
	lister := &mockLibraryLister{libraries: nil}

	handler := NewScanLibrariesHandler(lister, enq)
	if err := handler(context.Background(), nil); err != nil {
		t.Fatalf("handler: %v", err)
	}

	if len(enq.jobs) != 0 {
		t.Errorf("expected 0 enqueued jobs, got %d", len(enq.jobs))
	}
}

func TestScanLibrariesHandler_ListError(t *testing.T) {
	enq := &genericMockEnqueuer{}
	lister := &mockLibraryLister{err: errors.New("db error")}

	handler := NewScanLibrariesHandler(lister, enq)
	if err := handler(context.Background(), nil); err == nil {
		t.Fatal("expected error when ListLibraries fails")
	}
}

func TestScanLibrariesHandler_InvalidPaths(t *testing.T) {
	enq := &genericMockEnqueuer{}
	lister := &mockLibraryLister{
		libraries: []db.Library{
			{
				ID:        "lib1",
				Name:      "Bad",
				Paths:     `not valid json`,
				Monitored: true,
			},
			{
				ID:        "lib2",
				Name:      "Good",
				Paths:     `["/books/good"]`,
				Monitored: true,
			},
		},
	}

	handler := NewScanLibrariesHandler(lister, enq)
	if err := handler(context.Background(), nil); err != nil {
		t.Fatalf("handler should not fail on one bad library: %v", err)
	}

	// Only the good library should produce a job
	if got := len(enq.jobs); got != 1 {
		t.Errorf("expected 1 enqueued job, got %d", got)
	}
}

func TestScanLibrariesHandler_EnqueueError(t *testing.T) {
	enq := &genericMockEnqueuer{err: errors.New("redis unavailable")}
	lister := &mockLibraryLister{
		libraries: []db.Library{
			{
				ID:        "lib1",
				Name:      "Fiction",
				Paths:     `["/books/fiction"]`,
				Monitored: true,
			},
		},
	}

	handler := NewScanLibrariesHandler(lister, enq)
	// Enqueue errors should be logged but not cause the handler to fail
	if err := handler(context.Background(), nil); err != nil {
		t.Fatalf("handler should not fail on enqueue errors: %v", err)
	}
}
