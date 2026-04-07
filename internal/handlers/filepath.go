package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// isPathUnderRoot reports whether absPath is nested inside (or equal to) root.
// Both paths are resolved to absolute, cleaned, and symlink-resolved paths
// before the comparison so relative library roots and symlink escapes are
// handled correctly. When symlink resolution fails (e.g. the target does not
// exist yet) the function falls back to the cleaned absolute paths.
func isPathUnderRoot(absPath, root string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absRoot = filepath.Clean(absRoot)

	// Attempt symlink-aware canonicalization for both root and target.
	if resolved, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = resolved
	}
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolved
	} else if parent := filepath.Dir(absPath); parent != absPath {
		if resolvedParent, err := filepath.EvalSymlinks(parent); err == nil {
			absPath = filepath.Join(resolvedParent, filepath.Base(absPath))
		}
	}

	return absPath == absRoot || strings.HasPrefix(absPath, absRoot+string(filepath.Separator))
}

// isBookFilePathAllowed reports whether filePath resolves to a location that
// is nested inside one of the paths of any configured library. It returns
// false (with a nil error) when no library root contains the path.
func isBookFilePathAllowed(ctx context.Context, d *db.DB, filePath string) (bool, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return false, fmt.Errorf("resolve file path: %w", err)
	}
	absPath = filepath.Clean(absPath)
	// Resolve symlinks so a link targeting outside the library root is caught.
	// If the full path does not exist yet (common for new files), resolve the
	// parent directory and re-append the base name.
	if resolved, evalErr := filepath.EvalSymlinks(absPath); evalErr == nil {
		absPath = resolved
	} else if parent := filepath.Dir(absPath); parent != absPath {
		if resolvedParent, evalErr := filepath.EvalSymlinks(parent); evalErr == nil {
			absPath = filepath.Join(resolvedParent, filepath.Base(absPath))
		}
	}

	libraries, err := d.ListLibraries(ctx)
	if err != nil {
		return false, fmt.Errorf("list libraries: %w", err)
	}

	for _, lib := range libraries {
		var paths []string
		if jsonErr := json.Unmarshal([]byte(lib.Paths), &paths); jsonErr != nil {
			slog.WarnContext(ctx, "failed to parse library paths; skipping library during path validation",
				slog.String(otelkeys.LibraryID, lib.ID),
				slog.Any(otelkeys.Error, jsonErr),
			)
			continue
		}
		for _, p := range paths {
			if isPathUnderRoot(absPath, p) {
				return true, nil
			}
		}
	}
	return false, nil
}
