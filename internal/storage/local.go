package storage

import (
	"context"
	"fmt"
	"io"
	"os"
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
	return os.Open(path)
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

// Write implements Storage by creating or replacing the file at path with the
// contents of r.
func (l *LocalStorage) Write(ctx context.Context, path string, r io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	return nil
}

// Delete implements Storage by calling os.Remove.
func (l *LocalStorage) Delete(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Remove(path)
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
