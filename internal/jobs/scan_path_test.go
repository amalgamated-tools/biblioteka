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

func (m *mockEnqueuer) Enqueue(_ context.Context, name string, payload any, _ ...EnqueueOption) (string, error) {
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
	require.Len(t, mock.jobs, 7)

	// Verify all enqueued jobs target the process:file job
	for _, j := range mock.jobs {
		require.Equal(t, JobProcessFile, j.Name)
		require.NotEmpty(t, j.Payload.Path, "enqueued job has empty path")
		require.NotEmpty(t, j.Payload.FileType, "enqueued job has empty file type")
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

	require.Empty(t, mock.jobs)
}
