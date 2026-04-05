package jobs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestScanDirectory_EmptyPath verifies that an empty path is rejected.
func TestScanDirectory_EmptyPath(t *testing.T) {
	t.Parallel()

	err := ScanDirectory(context.Background(), &mockEnqueuer{}, ScanPathPayload{})
	require.Error(t, err, "expected error for empty path")
}

// TestScanDirectory_NonExistentPath verifies that a nonexistent path is rejected.
func TestScanDirectory_NonExistentPath(t *testing.T) {
	t.Parallel()

	err := ScanDirectory(context.Background(), &mockEnqueuer{}, ScanPathPayload{
		Path: "/nonexistent/path/that/does/not/exist",
	})
	require.Error(t, err, "expected error for nonexistent path")
}

// TestScanDirectory_NotADirectory verifies that a regular file path is rejected.
func TestScanDirectory_NotADirectory(t *testing.T) {
	t.Parallel()

	f, err := os.CreateTemp(t.TempDir(), "test-*.txt")
	require.NoError(t, err, "create temp file")
	_ = f.Close()

	scanErr := ScanDirectory(context.Background(), &mockEnqueuer{}, ScanPathPayload{Path: f.Name()})
	require.NotNil(t, scanErr)
}

// TestScanDirectory_EmptyDirectory verifies that an empty directory produces
// no enqueued jobs and no error.
func TestScanDirectory_EmptyDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	enqueuer := &mockEnqueuer{}

	require.NoError(t, ScanDirectory(context.Background(), enqueuer, ScanPathPayload{Path: dir}), "ScanDirectory() unexpected error")

	enqueuer.mu.Lock()
	defer enqueuer.mu.Unlock()
	require.Len(t, enqueuer.jobs, 0)
}

// TestScanDirectory_SupportedExtensions verifies that EPUB, MOBI, PDF, and
// AZW3 files are enqueued while unsupported files are skipped.
func TestScanDirectory_SupportedExtensions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	supported := map[string]string{
		"book.epub":   "epub",
		"novel.mobi":  "mobi",
		"manual.pdf":  "pdf",
		"kindle.azw3": "azw3",
	}
	skipped := []string{"image.jpg", "document.docx", "archive.zip", "readme.txt"}

	for name := range supported {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644), "write %s", name)
	}
	for _, name := range skipped {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644), "write %s", name)
	}

	enqueuer := &mockEnqueuer{}
	require.NoError(t, ScanDirectory(context.Background(), enqueuer, ScanPathPayload{Path: dir}), "ScanDirectory() unexpected error")

	enqueuer.mu.Lock()
	jobs := enqueuer.jobs
	enqueuer.mu.Unlock()

	require.Len(t, jobs, len(supported))

	for _, job := range jobs {
		wantType, ok := supported[filepath.Base(job.Payload.Path)]
		require.True(t, ok, "unexpected file enqueued: %q", job.Payload.Path)
		require.Equal(t, wantType, job.Payload.FileType)
	}
}

// TestScanDirectory_CaseInsensitiveExtensions verifies that file extensions
// are matched case-insensitively (e.g., .EPUB matches epub).
func TestScanDirectory_CaseInsensitiveExtensions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	files := []string{"BOOK.EPUB", "Novel.Mobi", "Manual.PDF", "Kindle.AZW3"}
	for _, name := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644), "write %s", name)
	}

	enqueuer := &mockEnqueuer{}
	require.NoError(t, ScanDirectory(context.Background(), enqueuer, ScanPathPayload{Path: dir}), "ScanDirectory() unexpected error")

	enqueuer.mu.Lock()
	count := len(enqueuer.jobs)
	enqueuer.mu.Unlock()

	require.Equal(t, len(files), count)
}

// TestScanDirectory_RecursiveWalk verifies that files in subdirectories are
// discovered and enqueued.
func TestScanDirectory_RecursiveWalk(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	subdir := filepath.Join(dir, "Author", "Series")
	require.NoError(t, os.MkdirAll(subdir, 0o755), "mkdir")

	files := []string{
		filepath.Join(dir, "root.epub"),
		filepath.Join(dir, "Author", "book1.epub"),
		filepath.Join(subdir, "book2.epub"),
	}
	for _, f := range files {
		require.NoError(t, os.WriteFile(f, []byte("data"), 0o644), "write %s", f)
	}

	enqueuer := &mockEnqueuer{}
	require.NoError(t, ScanDirectory(context.Background(), enqueuer, ScanPathPayload{Path: dir}), "ScanDirectory() unexpected error")

	enqueuer.mu.Lock()
	count := len(enqueuer.jobs)
	enqueuer.mu.Unlock()

	require.Equal(t, len(files), count)
}

// TestScanDirectory_EnqueueError verifies that enqueue errors are swallowed
// (logged as warnings) and processing continues without failing the scan.
func TestScanDirectory_EnqueueError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"a.epub", "b.epub", "c.epub"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644), "write %s", name)
	}

	failEnqueuer := &errEnqueuer{err: errors.New("queue full")}
	err := ScanDirectory(context.Background(), failEnqueuer, ScanPathPayload{Path: dir})
	require.NoError(t, err, "ScanDirectory() unexpected error when enqueue fails")
}

// TestScanDirectory_PayloadFields verifies that the enqueued job payload
// includes the correct FileName, FileType, and non-empty FilePath.
func TestScanDirectory_PayloadFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bookPath := filepath.Join(dir, "my-novel.epub")
	require.NoError(t, os.WriteFile(bookPath, []byte("epub content"), 0o644), "write")

	enqueuer := &mockEnqueuer{}
	err := ScanDirectory(context.Background(), enqueuer, ScanPathPayload{
		Path:      dir,
		LibraryID: "lib-123",
	})
	require.NoError(t, err, "ScanDirectory() error")

	enqueuer.mu.Lock()
	jobs := enqueuer.jobs
	enqueuer.mu.Unlock()

	require.Len(t, jobs, 1)
	p := jobs[0].Payload
	require.Equal(t, "my-novel.epub", p.FileName)
	require.Equal(t, "epub", p.FileType)
	require.NotEqual(t, "", p.Path)
	require.Equal(t, "lib-123", p.LibraryID)
}

// TestScanDirectory_ContextCancellation verifies that ScanDirectory respects
// context cancellation.
func TestScanDirectory_ContextCancellation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Create enough files to trigger the context check.
	for i := range 10 {
		name := filepath.Join(dir, fmt.Sprintf("sub%d", i), "book.epub")
		require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o755), "mkdir")
		require.NoError(t, os.WriteFile(name, []byte("data"), 0o644), "write")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := ScanDirectory(ctx, &mockEnqueuer{}, ScanPathPayload{Path: dir})
	require.Error(t, err, "expected context cancellation error")
	require.ErrorIs(t, err, context.Canceled)
}

// errEnqueuer is an Enqueuer that always returns the configured error.
type errEnqueuer struct {
	err error
}

func (e *errEnqueuer) Enqueue(_ context.Context, _ string, _ any) (string, error) {
	return "", e.err
}
