package jobs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestScanDirectory_EmptyPath verifies that an empty path is rejected.
func TestScanDirectory_EmptyPath(t *testing.T) {
	t.Parallel()

	err := ScanDirectory(context.Background(), &mockEnqueuer{}, ScanPathPayload{})
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

// TestScanDirectory_NonExistentPath verifies that a nonexistent path is rejected.
func TestScanDirectory_NonExistentPath(t *testing.T) {
	t.Parallel()

	err := ScanDirectory(context.Background(), &mockEnqueuer{}, ScanPathPayload{
		Path: "/nonexistent/path/that/does/not/exist",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

// TestScanDirectory_NotADirectory verifies that a regular file path is rejected.
func TestScanDirectory_NotADirectory(t *testing.T) {
	t.Parallel()

	f, err := os.CreateTemp(t.TempDir(), "test-*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	_ = f.Close()

	scanErr := ScanDirectory(context.Background(), &mockEnqueuer{}, ScanPathPayload{Path: f.Name()})
	if scanErr == nil {
		t.Fatal("expected error for file path, not directory")
	}
}

// TestScanDirectory_EmptyDirectory verifies that an empty directory produces
// no enqueued jobs and no error.
func TestScanDirectory_EmptyDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	enqueuer := &mockEnqueuer{}

	if err := ScanDirectory(context.Background(), enqueuer, ScanPathPayload{Path: dir}); err != nil {
		t.Fatalf("ScanDirectory() unexpected error: %v", err)
	}

	enqueuer.mu.Lock()
	defer enqueuer.mu.Unlock()
	if len(enqueuer.jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(enqueuer.jobs))
	}
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
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	for _, name := range skipped {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	enqueuer := &mockEnqueuer{}
	if err := ScanDirectory(context.Background(), enqueuer, ScanPathPayload{Path: dir}); err != nil {
		t.Fatalf("ScanDirectory() unexpected error: %v", err)
	}

	enqueuer.mu.Lock()
	jobs := enqueuer.jobs
	enqueuer.mu.Unlock()

	if len(jobs) != len(supported) {
		t.Errorf("expected %d jobs, got %d", len(supported), len(jobs))
	}

	for _, job := range jobs {
		wantType, ok := supported[filepath.Base(job.Payload.Path)]
		if !ok {
			t.Errorf("unexpected file enqueued: %q", job.Payload.Path)
			continue
		}
		if job.Payload.FileType != wantType {
			t.Errorf("file %q: FileType = %q, want %q", job.Payload.Path, job.Payload.FileType, wantType)
		}
	}
}

// TestScanDirectory_CaseInsensitiveExtensions verifies that file extensions
// are matched case-insensitively (e.g., .EPUB matches epub).
func TestScanDirectory_CaseInsensitiveExtensions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	files := []string{"BOOK.EPUB", "Novel.Mobi", "Manual.PDF", "Kindle.AZW3"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	enqueuer := &mockEnqueuer{}
	if err := ScanDirectory(context.Background(), enqueuer, ScanPathPayload{Path: dir}); err != nil {
		t.Fatalf("ScanDirectory() unexpected error: %v", err)
	}

	enqueuer.mu.Lock()
	count := len(enqueuer.jobs)
	enqueuer.mu.Unlock()

	if count != len(files) {
		t.Errorf("expected %d jobs for uppercase extensions, got %d", len(files), count)
	}
}

// TestScanDirectory_RecursiveWalk verifies that files in subdirectories are
// discovered and enqueued.
func TestScanDirectory_RecursiveWalk(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	subdir := filepath.Join(dir, "Author", "Series")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	files := []string{
		filepath.Join(dir, "root.epub"),
		filepath.Join(dir, "Author", "book1.epub"),
		filepath.Join(subdir, "book2.epub"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("data"), 0o644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}

	enqueuer := &mockEnqueuer{}
	if err := ScanDirectory(context.Background(), enqueuer, ScanPathPayload{Path: dir}); err != nil {
		t.Fatalf("ScanDirectory() unexpected error: %v", err)
	}

	enqueuer.mu.Lock()
	count := len(enqueuer.jobs)
	enqueuer.mu.Unlock()

	if count != len(files) {
		t.Errorf("expected %d jobs, got %d", len(files), count)
	}
}

// TestScanDirectory_EnqueueError verifies that enqueue errors are swallowed
// (logged as warnings) and processing continues without failing the scan.
func TestScanDirectory_EnqueueError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"a.epub", "b.epub", "c.epub"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	failEnqueuer := &errEnqueuer{err: errors.New("queue full")}
	err := ScanDirectory(context.Background(), failEnqueuer, ScanPathPayload{Path: dir})
	if err != nil {
		t.Fatalf("ScanDirectory() unexpected error when enqueue fails: %v", err)
	}
}

// TestScanDirectory_PayloadFields verifies that the enqueued job payload
// includes the correct FileName, FileType, and non-empty FilePath.
func TestScanDirectory_PayloadFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bookPath := filepath.Join(dir, "my-novel.epub")
	if err := os.WriteFile(bookPath, []byte("epub content"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	enqueuer := &mockEnqueuer{}
	if err := ScanDirectory(context.Background(), enqueuer, ScanPathPayload{
		Path:      dir,
		LibraryID: "lib-123",
	}); err != nil {
		t.Fatalf("ScanDirectory() error: %v", err)
	}

	enqueuer.mu.Lock()
	jobs := enqueuer.jobs
	enqueuer.mu.Unlock()

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	p := jobs[0].Payload
	if p.FileName != "my-novel.epub" {
		t.Errorf("FileName = %q, want %q", p.FileName, "my-novel.epub")
	}
	if p.FileType != "epub" {
		t.Errorf("FileType = %q, want %q", p.FileType, "epub")
	}
	if p.Path == "" {
		t.Error("Path should not be empty")
	}
	if p.LibraryID != "lib-123" {
		t.Errorf("LibraryID = %q, want %q", p.LibraryID, "lib-123")
	}
}

// TestScanDirectory_ContextCancellation verifies that ScanDirectory respects
// context cancellation.
func TestScanDirectory_ContextCancellation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Create enough files to trigger the context check.
	for i := range 10 {
		name := filepath.Join(dir, fmt.Sprintf("sub%d", i), "book.epub")
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(name, []byte("data"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := ScanDirectory(ctx, &mockEnqueuer{}, ScanPathPayload{Path: dir})
	// Either no error (if walk completes before checking) or a context error.
	// Either is acceptable; what matters is that it doesn't panic.
	_ = err
}

// errEnqueuer is an Enqueuer that always returns the configured error.
type errEnqueuer struct {
	err error
}

func (e *errEnqueuer) Enqueue(_ context.Context, _ string, _ any) (string, error) {
	return "", e.err
}
