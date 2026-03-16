package testutils

import (
	"path/filepath"
	"runtime"
)

var _, b, _, _ = runtime.Caller(0)

func GetProjectRoot() string {
	return filepath.Join(filepath.Dir(b), "../..") //nolint:gocritic // This is a safe operation.
}
