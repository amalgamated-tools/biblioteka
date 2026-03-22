//go:build !linux

package organize

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

func renameNoReplace(ctx context.Context, oldPath, newPath string) error {
	if err := os.Link(oldPath, newPath); err != nil {
		if os.IsExist(err) {
			slog.WarnContext(
				ctx,
				"file already exists at destination",
				slog.String(otelkeys.NewPath, newPath),
			)
			return os.ErrExist
		}
		if isCrossDeviceRenameError(err) {
			slog.DebugContext(
				ctx,
				"cross-filesystem rename detected, falling back to copy+remove",
				slog.String(otelkeys.OldPath, oldPath),
				slog.String(otelkeys.NewPath, newPath),
			)
			return fmt.Errorf("cross-filesystem rename from %s to %s: %w", oldPath, newPath, err)
		}
		slog.ErrorContext(
			ctx,
			"failed to create hard link for rename fallback",
			slog.String(otelkeys.OldPath, oldPath),
			slog.String(otelkeys.NewPath, newPath),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("create hard link from %s to %s: %w", oldPath, newPath, err)
	}
	if err := os.Remove(oldPath); err != nil {
		_ = os.Remove(newPath)
		slog.ErrorContext(
			ctx,
			"failed to remove old file after creating hard link during rename fallback",
			slog.String(otelkeys.OldPath, oldPath),
			slog.String(otelkeys.NewPath, newPath),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("remove old file %s: %w", oldPath, err)
	}

	return nil
}
