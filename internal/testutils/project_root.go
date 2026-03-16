package testutils

import (
	"path/filepath"
	"runtime"
)

func GetProjectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("testutils: runtime.Caller(0) failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(filename), "../..")) //nolint:gocritic // This is a safe operation.
}
