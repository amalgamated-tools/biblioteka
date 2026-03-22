//go:build linux

package organize

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/sys/unix"
)

func renameNoReplace(ctx context.Context, oldPath, newPath string) error {
	err := unix.Renameat2(unix.AT_FDCWD, oldPath, unix.AT_FDCWD, newPath, unix.RENAME_NOREPLACE)
	if err == nil {
		slog.DebugContext(
			ctx,
			"renamed file without replacement",
			slog.String(otelkeys.OldPath, oldPath),
			slog.String(otelkeys.NewPath, newPath),
		)
		return nil
	}
	if err == unix.EEXIST {
		slog.WarnContext(
			ctx,
			"file already exists at destination",
			slog.String(otelkeys.NewPath, newPath),
		)
		return os.ErrExist
	}
	// Cross-filesystem: propagate EXDEV so the caller can use copy+remove.
	if err == unix.EXDEV {
		slog.DebugContext(
			ctx,
			"cross-filesystem rename detected, falling back to copy+remove",
			slog.String(otelkeys.OldPath, oldPath),
			slog.String(otelkeys.NewPath, newPath),
		)
		return fmt.Errorf("cross-filesystem rename from %s to %s: %w", oldPath, newPath, err)
	}
	// Fallback for kernels/filesystems that do not support RENAME_NOREPLACE.
	// Best-effort: stat+rename has a TOCTOU window but there is no atomic alternative.
	if err == unix.ENOSYS || err == unix.EINVAL {
		if _, statErr := os.Stat(newPath); statErr == nil {
			slog.WarnContext(
				ctx,
				"file already exists at destination (fallback)",
				slog.String(otelkeys.NewPath, newPath),
			)
			return os.ErrExist
		} else if !os.IsNotExist(statErr) {
			slog.ErrorContext(
				ctx,
				"failed to stat destination file during rename fallback",
				slog.String(otelkeys.NewPath, newPath),
				slog.Any(otelkeys.Error, statErr),
			)
			return fmt.Errorf("stat destination file %s: %w", newPath, statErr)
		}
		if renameErr := os.Rename(oldPath, newPath); renameErr != nil {
			slog.ErrorContext(
				ctx,
				"fallback rename failed",
				slog.String(otelkeys.OldPath, oldPath),
				slog.String(otelkeys.NewPath, newPath),
				slog.Any(otelkeys.Error, renameErr),
			)
			return fmt.Errorf("fallback rename %s to %s: %w", oldPath, newPath, renameErr)
		}
		slog.DebugContext(
			ctx,
			"renamed file without replacement (fallback)",
			slog.String(otelkeys.OldPath, oldPath),
			slog.String(otelkeys.NewPath, newPath),
		)
		return nil
	}
	return fmt.Errorf("rename %s to %s: %w", oldPath, newPath, err)
}
