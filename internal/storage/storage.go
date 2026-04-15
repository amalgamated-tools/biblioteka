// Package storage defines an abstract file-storage interface.
// It is the foundation for future storage backends such as S3:
// callers that depend on Storage can switch backends without code changes.
package storage

import (
	"context"
	"io"
	"time"
)

// FileInfo holds metadata about a file or directory returned by Storage.Stat
// or Storage.List.
type FileInfo struct {
	Name    string
	Size    int64
	IsDir   bool
	ModTime time.Time
}

// Storage is an abstract interface over a file-storage backend.
// Concrete implementations (e.g. a local filesystem or an S3-compatible
// object store) satisfy this interface.
//
// All methods accept a context.Context for cancellation and deadline
// propagation. Implementations must be safe for concurrent use by multiple
// goroutines.
type Storage interface {
	// Open opens the named file for reading.
	Open(ctx context.Context, path string) (io.ReadCloser, error)

	// Stat returns metadata for the file or directory at path.
	Stat(ctx context.Context, path string) (FileInfo, error)

	// Write creates or replaces the file at path with the contents of r.
	Write(ctx context.Context, path string, r io.Reader) error

	// Delete removes the file at path.
	Delete(ctx context.Context, path string) error

	// List returns FileInfo for every entry directly under prefix (non-recursive).
	// For local storage, prefix is a directory path. For S3, prefix is a key
	// prefix (e.g. "books/2024/") that must end with "/" to scope results to a
	// single "directory" level.
	List(ctx context.Context, prefix string) ([]FileInfo, error)
}
