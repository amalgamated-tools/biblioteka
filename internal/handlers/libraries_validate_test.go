package handlers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestValidatePathsWith_BlockingStat verifies that validatePathsWith returns a
// context.DeadlineExceeded-wrapped error when the stat function hangs and the
// supplied context times out before it completes.
func TestValidatePathsWith_BlockingStat(t *testing.T) {
	t.Parallel()

	// done is closed when the test ends so the goroutine inside validatePathsWith
	// can unblock and exit cleanly (no goroutine leak).
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })

	blockingStat := func(string) (os.FileInfo, error) {
		<-done
		return nil, os.ErrNotExist
	}

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	err := validatePathsWith(ctx, []string{"/any/path"}, blockingStat)
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestValidatePathsWith_ValidDir verifies that validatePathsWith returns nil for
// a real directory.
func TestValidatePathsWith_ValidDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := validatePathsWith(t.Context(), []string{dir}, os.Stat)
	require.NoError(t, err)
}

// TestValidatePathsWith_NonexistentPath verifies that validatePathsWith returns
// a pathValidationError for a path that does not exist.
func TestValidatePathsWith_NonexistentPath(t *testing.T) {
	t.Parallel()

	err := validatePathsWith(t.Context(), []string{"/nonexistent/path/xyz"}, os.Stat)
	require.Error(t, err)

	var pve *pathValidationError
	require.ErrorAs(t, err, &pve)
}

// TestValidatePathsWith_FileNotDir verifies that validatePathsWith returns a
// pathValidationError when a path points to a regular file instead of a directory.
func TestValidatePathsWith_FileNotDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "notadir")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	verr := validatePathsWith(t.Context(), []string{f.Name()}, os.Stat)
	require.Error(t, verr)

	var pve *pathValidationError
	require.ErrorAs(t, verr, &pve)
}
