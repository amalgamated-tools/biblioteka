//go:build linux

package organize

import (
	"os"

	"golang.org/x/sys/unix"
)

func renameNoReplace(oldPath, newPath string) error {
	err := unix.Renameat2(unix.AT_FDCWD, oldPath, unix.AT_FDCWD, newPath, unix.RENAME_NOREPLACE)
	if err == nil {
		return nil
	}
	if err == unix.EEXIST {
		return os.ErrExist
	}
	// Cross-filesystem: propagate EXDEV so the caller can use copy+remove.
	if err == unix.EXDEV {
		return err
	}
	// Fallback for kernels/filesystems that do not support RENAME_NOREPLACE.
	// Best-effort: stat+rename has a TOCTOU window but there is no atomic alternative.
	if err == unix.ENOSYS || err == unix.EINVAL {
		if _, statErr := os.Stat(newPath); statErr == nil {
			return os.ErrExist
		} else if !os.IsNotExist(statErr) {
			return statErr
		}
		return os.Rename(oldPath, newPath)
	}
	return err
}
