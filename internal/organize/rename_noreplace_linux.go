//go:build linux

package organize

import (
	"errors"
	"os"
	"syscall"
)

func renameNoReplace(oldPath, newPath string) error {
	err := syscall.Renameat2(syscall.AT_FDCWD, oldPath, syscall.AT_FDCWD, newPath, syscall.RENAME_NOREPLACE)
	if errors.Is(err, syscall.EEXIST) {
		return os.ErrExist
	}

	return err
}
