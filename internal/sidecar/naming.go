package sidecar

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

func validateBaseName(baseName string) error {
	if baseName == "" {
		return nil
	}
	if baseName == "." || baseName == ".." || baseName != filepath.Base(baseName) || strings.ContainsAny(baseName, `/\`) {
		return fmt.Errorf("invalid sidecar base name %q", baseName)
	}

	return nil
}

func sidecarTarget(bookFilePath, organizationType string) (dir string, baseName string, err error) {
	if bookFilePath == "" {
		return "", "", fmt.Errorf("book file path is required")
	}

	dir = filepath.Dir(bookFilePath)
	if organizationType == db.LibraryOrganizationBookPerFile {
		fileName := filepath.Base(bookFilePath)
		baseName = strings.TrimSuffix(fileName, filepath.Ext(fileName))
	}
	if err := validateBaseName(baseName); err != nil {
		return "", "", err
	}

	return dir, baseName, nil
}
