package jobs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockEnqueuer records all enqueued jobs for test assertions.
type mockEnqueuer struct {
	mu   sync.Mutex
	jobs []enqueuedJob
}

type enqueuedJob struct {
	Name    string
	Payload ProcessFilePayload
}

func (m *mockEnqueuer) Enqueue(_ context.Context, name string, payload any) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	var p ProcessFilePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs = append(m.jobs, enqueuedJob{Name: name, Payload: p})
	return "mock-id", nil
}

func TestScanPathHandler(t *testing.T) {
	mock := &mockEnqueuer{}

	// Create temp directory with test files
	dir := t.TempDir()

	testFiles := map[string]bool{
		"My Book.epub":      true,
		"Another Book.mobi": true,
		"Third Book.pdf":    true,
		"Kindle Book.azw3":  true,
		"not-a-book.txt":    false,
		"image.jpg":         false,
		"UPPERCASE.EPUB":    true,
		"MixedCase.Mobi":    true,
	}

	for name := range testFiles {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("test content"), 0o644), "write file %s", name)
	}

	// Create a subdirectory with another book
	subdir := filepath.Join(dir, "subdir")
	require.NoError(t, os.MkdirAll(subdir, 0o755), "mkdir")
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "Nested.pdf"), []byte("nested content"), 0o644), "write nested file")

	// Run the handler
	handler := NewScanPathHandler(mock)
	payload, err := json.Marshal(ScanPathPayload{Path: dir})
	require.NoError(t, err, "marshal payload")

	require.NoError(t, handler(t.Context(), payload), "handler")

	// 6 matching files: My Book.epub, Another Book.mobi, Third Book.pdf, Kindle Book.azw3, UPPERCASE.EPUB, MixedCase.Mobi
	// Plus 1 nested: Nested.pdf = 7 total
	if got := len(mock.jobs); got != 7 {
		t.Errorf("expected 7 enqueued jobs, got %d", got)
		for _, j := range mock.jobs {
			t.Logf("  job: %s %s", j.Name, j.Payload.FileName)
		}
	}

	// Verify all enqueued jobs target the process:file job
	for _, j := range mock.jobs {
		if j.Name != JobProcessFile {
			t.Errorf("expected job name %q, got %q", JobProcessFile, j.Name)
		}
		if j.Payload.Path == "" {
			t.Error("enqueued job has empty path")
		}
		if j.Payload.FileType == "" {
			t.Error("enqueued job has empty file type")
		}
	}
}

func TestScanPathHandler_EmptyPath(t *testing.T) {
	mock := &mockEnqueuer{}
	handler := NewScanPathHandler(mock)

	payload, err := json.Marshal(ScanPathPayload{Path: ""})
	require.NoError(t, err, "marshal")
	err = handler(t.Context(), payload)
	require.Error(t, err, "expected error for empty path")
}

func TestScanPathHandler_NonexistentPath(t *testing.T) {
	mock := &mockEnqueuer{}
	handler := NewScanPathHandler(mock)

	payload, err := json.Marshal(ScanPathPayload{Path: "/nonexistent/path/that/does/not/exist"})
	require.NoError(t, err, "marshal")
	err = handler(t.Context(), payload)
	require.Error(t, err, "expected error for nonexistent path")
}

func TestScanPathHandler_EmptyDirectory(t *testing.T) {
	mock := &mockEnqueuer{}
	handler := NewScanPathHandler(mock)

	dir := t.TempDir()
	payload, err := json.Marshal(ScanPathPayload{Path: dir})
	require.NoError(t, err, "marshal")

	require.NoError(t, handler(t.Context(), payload), "handler")

	if len(mock.jobs) != 0 {
		t.Errorf("expected 0 enqueued jobs, got %d", len(mock.jobs))
	}
}
