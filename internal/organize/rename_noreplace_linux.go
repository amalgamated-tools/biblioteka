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
	// Fallback for kernels/filesystems that do not support renameat2(RENAME_NOREPLACE)
	if err == unix.ENOSYS || err == unix.EINVAL || err == unix.EXDEV {
		// Best-effort no-replace: fail if destination already exists
		if _, statErr := os.Stat(newPath); statErr == nil {
			return os.ErrExist
		} else if !os.IsNotExist(statErr) {
			return statErr
		}

		// Destination does not exist (to the best of our knowledge); attempt rename
		return os.Rename(oldPath, newPath)
	}
	return err
}
