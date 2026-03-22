package organize

import (
	"errors"
	"os"
	"syscall"
)

func isCrossDeviceRenameError(err error) bool {
	return errors.Is(err, syscall.EXDEV)
}

func isBenignCleanupRemoveError(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTEMPTY)
}
