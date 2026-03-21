//go:build !linux

package organize

import (
	"os"
)

func renameNoReplace(oldPath, newPath string) error {
	if err := os.Link(oldPath, newPath); err != nil {
		if os.IsExist(err) {
			return os.ErrExist
		}
		return err
	}
	if err := os.Remove(oldPath); err != nil {
		_ = os.Remove(newPath)
		return err
	}

	return nil
}
