//go:build linux

package organize

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	atFDCWD             = -100
	renameNoReplaceFlag = 1
)

func renameNoReplace(oldPath, newPath string) error {
	oldPtr, err := syscall.BytePtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPtr, err := syscall.BytePtrFromString(newPath)
	if err != nil {
		return err
	}

	_, _, errno := syscall.Syscall6(
		syscall.SYS_RENAMEAT2,
		uintptr(atFDCWD),
		uintptr(unsafe.Pointer(oldPtr)),
		uintptr(atFDCWD),
		uintptr(unsafe.Pointer(newPtr)),
		uintptr(renameNoReplaceFlag),
		0,
	)
	if errno == 0 {
		return nil
	}
	if errno == syscall.EEXIST {
		return os.ErrExist
	}
	return errno
}
