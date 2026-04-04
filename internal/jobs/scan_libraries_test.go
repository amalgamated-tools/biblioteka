package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

// mockLibraryLister records calls to ListLibraries and returns a preset list.
type mockLibraryLister struct {
	libraries []db.Library
	err       error
}

func (m *mockLibraryLister) ListLibraries(_ context.Context) ([]db.Library, error) {
	return m.libraries, m.err
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
	if err := handler(t.Context(), nil); err != nil {
		require.NoError(t, err, "handler")
	}

	// Expect 2 scan:library jobs (lib1 and lib2; lib3 is not monitored)
	if got := len(enq.jobs); got != 2 {
		require.Failf(t, "failed", "expected 2 enqueued jobs, got %d", got)
	}

	wantJobs := map[string][]string{
		"lib1": {"/books/fiction", "/books/scifi"},
		"lib2": {"/books/nonfiction"},
	}
	for _, j := range enq.jobs {
		if j.Name != JobScanLibrary {
			t.Errorf("expected job name %q, got %q", JobScanLibrary, j.Name)
		}
		var p ScanLibraryPayload
		if err := json.Unmarshal(j.Payload, &p); err != nil {
			t.Errorf("unmarshal payload: %v", err)
			continue
		}
		wantPaths, ok := wantJobs[p.LibraryID]
		if !ok {
			t.Errorf("unexpected library_id %q", p.LibraryID)
			continue
		}
		if !slices.Equal(p.Paths, wantPaths) {
			t.Errorf("library %q paths = %v, want %v", p.LibraryID, p.Paths, wantPaths)
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
	if err := handler(t.Context(), nil); err != nil {
		require.NoError(t, err, "handler")
	}

	if len(enq.jobs) != 0 {
		t.Errorf("expected 0 enqueued jobs, got %d", len(enq.jobs))
	}
}

func TestScanLibrariesHandler_EmptyLibraryList(t *testing.T) {
	enq := &genericMockEnqueuer{}
	lister := &mockLibraryLister{libraries: nil}

	handler := NewScanLibrariesHandler(lister, enq)
	if err := handler(t.Context(), nil); err != nil {
		require.NoError(t, err, "handler")
	}

	if len(enq.jobs) != 0 {
		t.Errorf("expected 0 enqueued jobs, got %d", len(enq.jobs))
	}
}

func TestScanLibrariesHandler_ListError(t *testing.T) {
	enq := &genericMockEnqueuer{}
	lister := &mockLibraryLister{err: errors.New("db error")}

	handler := NewScanLibrariesHandler(lister, enq)
	if err := handler(t.Context(), nil); err == nil {
		require.Fail(t, "expected error when ListLibraries fails")
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
	if err := handler(t.Context(), nil); err != nil {
		require.NoError(t, err, "handler should not fail on one bad library")
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
	if err := handler(t.Context(), nil); err != nil {
		require.NoError(t, err, "handler should not fail on enqueue errors")
	}
}
