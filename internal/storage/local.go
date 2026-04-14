package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Compile-time assertion that LocalStorage implements Storage.
var _ Storage = (*LocalStorage)(nil)

// LocalStorage is a Storage implementation backed by the local filesystem.
type LocalStorage struct{}

// NewLocalStorage returns a new LocalStorage.
func NewLocalStorage() *LocalStorage {
	return &LocalStorage{}
}

// Open implements Storage by calling os.Open.
func (l *LocalStorage) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return f, nil
}

// Stat implements Storage by calling os.Stat.
func (l *LocalStorage) Stat(ctx context.Context, path string) (FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return FileInfo{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		IsDir:   info.IsDir(),
		ModTime: info.ModTime(),
	}, nil
}

// Write implements Storage by atomically creating or replacing the file at path
// with the contents of r. It writes to a temporary file and renames on success,
// so a crash or mid-write error never leaves a partial file at path.
// The parent directory must already exist.
func (l *LocalStorage) Write(ctx context.Context, path string, r io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-write-*")
	if err != nil {
		return fmt.Errorf("create temp for %s: %w", path, err)
	}
	tmpName := tmp.Name()

	if _, err := io.Copy(tmp, newContextReader(ctx, r)); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename to %s: %w", path, err)
	}
	return nil
}

// Delete implements Storage by calling os.Remove.
func (l *LocalStorage) Delete(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete %s: %w", path, err)
	}
	return nil
}

// List implements Storage by reading the directory at prefix and returning one
// FileInfo per entry (non-recursive).
func (l *LocalStorage) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(prefix)
	if err != nil {
		return nil, err
	}
	infos := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		info, err := e.Info()
		if err != nil {
			return nil, fmt.Errorf("stat entry %s: %w", e.Name(), err)
		}
		infos = append(infos, FileInfo{
			Name:    e.Name(),
			Size:    info.Size(),
			IsDir:   e.IsDir(),
			ModTime: info.ModTime(),
		})
	}
	return infos, nil
}

// contextReader wraps an io.Reader and checks ctx.Err() before each Read,
// so that context cancellation is propagated into long-running io.Copy calls.
type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func newContextReader(ctx context.Context, r io.Reader) *contextReader {
	return &contextReader{ctx: ctx, r: r}
}

func (cr *contextReader) Read(p []byte) (int, error) {
	if err := cr.ctx.Err(); err != nil {
		return 0, err
	}
	return cr.r.Read(p)
}
