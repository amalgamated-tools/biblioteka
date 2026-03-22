package organize

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ReorganizeFile moves a book file into the canonical Author/Title/ directory
// structure under libraryRoot. If the file is already in the correct location,
// it returns the original path unchanged. After moving, empty source
// directories are cleaned up (up to but not including libraryRoot).
//
// Returns the new absolute path of the file.
func ReorganizeFile(ctx context.Context, filePath, libraryRoot, author, title string) (string, error) {
	if author == "" || title == "" {
		slog.DebugContext(ctx, "organize: skipping reorganization due to empty author or title")
		return filePath, nil
	}

	// Sanitize directory names: remove path separators and leading dots.
	safeAuthor := sanitizeDirName(author)
	safeTitle := sanitizeDirName(title)
	if safeAuthor == "" || safeTitle == "" {
		return filePath, nil
	}

	filename := filepath.Base(filePath)
	targetDir := filepath.Join(libraryRoot, safeAuthor, safeTitle)
	targetPath := filepath.Join(targetDir, filename)

	// Defense-in-depth: verify the target path is still inside libraryRoot.
	// Author/title may come from untrusted epub metadata.
	relCheck, err := filepath.Rel(libraryRoot, targetPath)
	if err != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(os.PathSeparator)) {
		attrs := []slog.Attr{
			slog.String(otelkeys.TargetPath, targetPath),
			slog.String(otelkeys.LibraryRoot, libraryRoot),
			slog.String(otelkeys.RelPath, relCheck),
		}
		if err != nil {
			attrs = append(attrs, slog.Any(otelkeys.Error, err))
		}
		slog.LogAttrs(
			ctx,
			slog.LevelWarn,
			"organize: target path escapes library root, skipping reorganization",
			attrs...,
		)
		return "", fmt.Errorf("target path %q escapes library root %q", targetPath, libraryRoot)
	}

	// Already in the right place.
	if filepath.Clean(filePath) == filepath.Clean(targetPath) {
		slog.DebugContext(
			ctx,
			"organize: file already in target location, skipping reorganization",
			slog.String(otelkeys.FilePath, filePath),
			slog.String(otelkeys.TargetPath, targetPath),
		)
		return filePath, nil
	}

	// Create target directory.
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		slog.ErrorContext(
			ctx,
			"organize: failed to create target directory",
			slog.String(otelkeys.TargetPath, targetPath),
			slog.String(otelkeys.LibraryRoot, libraryRoot),
			slog.Any(otelkeys.Error, err),
		)
		return "", fmt.Errorf("create target directory %s: %w", targetDir, err)
	}

	return moveFileIntoLibrary(ctx, filePath, targetPath, libraryRoot)
}

// cleanEmptyDirs removes empty directories from dir up to (but not including)
// stopAt. Stops as soon as a non-empty directory is encountered.
func cleanEmptyDirs(ctx context.Context, dir, stopAt string) {
	stopAt = filepath.Clean(stopAt)
	for {
		dir = filepath.Clean(dir)

		// Ensure dir is a strict descendant of stopAt.
		rel, err := filepath.Rel(stopAt, dir)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"organize: failed to compute relative path during cleanup, stopping cleanup",
				slog.String(otelkeys.Directory, dir),
				slog.String(otelkeys.StopAt, stopAt),
				slog.Any(otelkeys.Error, err),
			)
			return
		}
		// If dir is the same as stopAt, or outside/above it, stop.
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
			slog.DebugContext(
				ctx,
				"organize: reached stop directory during cleanup, stopping cleanup",
				slog.String(otelkeys.Directory, dir),
				slog.String(otelkeys.StopAt, stopAt),
			)
			return
		}

		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(dir); err != nil {
			slog.ErrorContext(
				ctx,
				"organize: failed to remove empty directory during cleanup",
				slog.String(otelkeys.Directory, dir),
				slog.String(otelkeys.StopAt, stopAt),
				slog.Any(otelkeys.Error, err),
			)
			return
		}
		dir = filepath.Dir(dir)
	}
}

func copyFile(ctx context.Context, src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"organize: failed to open source file for copying",
			slog.String(otelkeys.Source, src),
			slog.String(otelkeys.Destination, dst),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("open source file %s: %w", src, err)
	}
	defer in.Close()

	// Capture source file metadata so we can preserve it on the destination.
	srcInfo, err := in.Stat()
	if err != nil {
		slog.ErrorContext(
			ctx,
			"organize: failed to stat source file",
			slog.String(otelkeys.Source, src),
			slog.String(otelkeys.Destination, dst),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("stat source file %s: %w", src, err)
	}
	srcMode := srcInfo.Mode()
	srcModTime := srcInfo.ModTime()

	dstDir := filepath.Dir(dst)
	out, err := os.CreateTemp(dstDir, ".biblioteka-copy-*")
	if err != nil {
		slog.ErrorContext(
			ctx,
			"organize: failed to create temp file for copying",
			slog.String(otelkeys.Source, src),
			slog.String(otelkeys.Destination, dst),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("create temp file in %s: %w", dstDir, err)
	}
	tmpName := out.Name()

	// Ensure the temp file has the same permissions as the source file, so that
	// the copy behaves like a rename from the user's perspective.
	if err := out.Chmod(srcMode.Perm()); err != nil {
		_ = out.Close()
		slog.ErrorContext(
			ctx,
			"organize: failed to set permissions on temp file during copying",
			slog.String(otelkeys.Source, src),
			slog.String(otelkeys.Destination, dst),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("chmod temp file %s: %w", tmpName, err)
	}

	defer func() {
		// Ensure the temp file is cleaned up on any error.
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		slog.ErrorContext(
			ctx,
			"organize: failed to copy file contents",
			slog.String(otelkeys.Source, src),
			slog.String(otelkeys.Destination, dst),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("copy file %s to %s: %w", src, dst, err)
	}

	// Close the temp file before renaming to ensure contents are flushed and to
	// avoid rename failures on platforms like Windows.
	if cerr := out.Close(); cerr != nil {
		slog.ErrorContext(
			ctx,
			"organize: failed to close temp file after copying",
			slog.String(otelkeys.Source, src),
			slog.String(otelkeys.Destination, dst),
			slog.Any(otelkeys.Error, cerr),
		)
		return fmt.Errorf("close temp file %s: %w", tmpName, cerr)
	}

	// Preserve modification time (and use it for access time as well) so the
	// copied file closely matches the original's metadata.
	if err = os.Chtimes(tmpName, srcModTime, srcModTime); err != nil {
		slog.ErrorContext(
			ctx,
			"organize: failed to set modification time on temp file during copying",
			slog.String(otelkeys.Source, src),
			slog.String(otelkeys.Destination, dst),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("set modification time on temp file %s: %w", tmpName, err)
	}

	if err = renameNoReplace(ctx, tmpName, dst); err != nil {
		slog.ErrorContext(
			ctx,
			"organize: failed to rename temp file to destination during copying",
			slog.String(otelkeys.Source, src),
			slog.String(otelkeys.Destination, dst),
			slog.Any(otelkeys.Error, err),
		)
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("destination file %q already exists", dst)
		}
		return fmt.Errorf("rename temp file %s to %s: %w", tmpName, dst, err)
	}

	return nil
}

// ReorganizeFileFlat moves a book file into an Author/ directory structure
// under libraryRoot (flat, no title subdirectory). If the file is already in
// the correct location, it returns the original path unchanged. After moving,
// empty source directories are cleaned up (up to but not including libraryRoot).
//
// Returns the new absolute path of the file.
func ReorganizeFileFlat(ctx context.Context, filePath, libraryRoot, author string) (string, error) {
	if author == "" {
		slog.DebugContext(ctx, "organize: skipping reorganization due to empty author")
		return filePath, nil
	}

	safeAuthor := sanitizeDirName(author)
	if safeAuthor == "" {
		slog.DebugContext(ctx, "organize: skipping reorganization due to sanitized empty author")
		return filePath, nil
	}

	filename := filepath.Base(filePath)
	targetDir := filepath.Join(libraryRoot, safeAuthor)
	targetPath := filepath.Join(targetDir, filename)

	// Defense-in-depth: verify the target path is still inside libraryRoot.
	relCheck, err := filepath.Rel(libraryRoot, targetPath)
	if err != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(os.PathSeparator)) {
		attrs := []slog.Attr{
			slog.String(otelkeys.TargetPath, targetPath),
			slog.String(otelkeys.LibraryRoot, libraryRoot),
			slog.String(otelkeys.RelPath, relCheck),
		}
		if err != nil {
			attrs = append(attrs, slog.Any(otelkeys.Error, err))
		}
		slog.LogAttrs(
			ctx,
			slog.LevelWarn,
			"organize: target path escapes library root, skipping reorganization",
			attrs...,
		)
		return "", fmt.Errorf("target path %q escapes library root %q", targetPath, libraryRoot)
	}

	// Already in the right place.
	if filepath.Clean(filePath) == filepath.Clean(targetPath) {
		slog.DebugContext(
			ctx,
			"organize: file already in target location, skipping reorganization",
			slog.String(otelkeys.FilePath, filePath),
			slog.String(otelkeys.TargetPath, targetPath),
		)
		return filePath, nil
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		slog.ErrorContext(
			ctx,
			"organize: failed to create target directory",
			slog.String(otelkeys.TargetPath, targetPath),
			slog.String(otelkeys.LibraryRoot, libraryRoot),
			slog.Any(otelkeys.Error, err),
		)
		return "", fmt.Errorf("create target directory %s: %w", targetDir, err)
	}

	return moveFileIntoLibrary(ctx, filePath, targetPath, libraryRoot)
}

func moveFileIntoLibrary(ctx context.Context, filePath, targetPath, libraryRoot string) (string, error) {
	if err := renameNoReplace(ctx, filePath, targetPath); err == nil {
		slog.DebugContext(
			ctx,
			"organize: successfully moved file with rename",
			slog.String(otelkeys.FilePath, filePath),
			slog.String(otelkeys.TargetPath, targetPath),
		)
		cleanEmptyDirs(ctx, filepath.Dir(filePath), libraryRoot)
		return targetPath, nil
	} else if errors.Is(err, os.ErrExist) {
		slog.WarnContext(
			ctx,
			"organize: target file already exists, skipping reorganization",
			slog.String(otelkeys.FilePath, filePath),
			slog.String(otelkeys.TargetPath, targetPath),
		)
		return "", fmt.Errorf("target file already exists: %s", targetPath)
	} else if !errors.Is(err, syscall.EXDEV) {
		slog.ErrorContext(
			ctx,
			"organize: failed to rename file",
			slog.String(otelkeys.FilePath, filePath),
			slog.String(otelkeys.TargetPath, targetPath),
			slog.Any(otelkeys.Error, err),
		)
		return "", fmt.Errorf("rename %s to %s: %w", filePath, targetPath, err)
	}

	// Cross-filesystem fallback: copy then remove.
	if err := copyFile(ctx, filePath, targetPath); err != nil {
		slog.ErrorContext(
			ctx,
			"organize: failed to copy file during cross-filesystem move",
			slog.String(otelkeys.FilePath, filePath),
			slog.String(otelkeys.TargetPath, targetPath),
			slog.Any(otelkeys.Error, err),
		)
		return "", fmt.Errorf("copy file to %s: %w", targetPath, err)
	}
	if err := os.Remove(filePath); err != nil {
		slog.ErrorContext(
			ctx,
			"organize: failed to remove original file after copy during cross-filesystem move",
			slog.String(otelkeys.FilePath, filePath),
			slog.String(otelkeys.TargetPath, targetPath),
			slog.Any(otelkeys.Error, err),
		)
		return targetPath, fmt.Errorf("remove original file %s after copy to %s: %w", filePath, targetPath, err)
	}

	slog.DebugContext(
		ctx,
		"organize: successfully moved file with cross-filesystem copy fallback",
		slog.String(otelkeys.FilePath, filePath),
		slog.String(otelkeys.TargetPath, targetPath),
	)
	cleanEmptyDirs(ctx, filepath.Dir(filePath), libraryRoot)
	return targetPath, nil
}

// TargetPathFlat returns the canonical target file path that
// ReorganizeFileFlat would produce for the given inputs, without actually
// moving anything. Returns an empty string if author is empty or sanitizes
// to empty.
func TargetPathFlat(ctx context.Context, filePath, libraryRoot, author string) string {
	if author == "" {
		slog.DebugContext(ctx, "organize: skipping reorganization due to empty author")
		return ""
	}
	safeAuthor := sanitizeDirName(author)
	if safeAuthor == "" {
		slog.DebugContext(ctx, "organize: skipping reorganization due to sanitized empty author")
		return ""
	}
	return filepath.Join(libraryRoot, safeAuthor, filepath.Base(filePath))
}

// TargetPath returns the canonical target file path that ReorganizeFile would
// produce for the given inputs, without actually moving anything. Returns an
// empty string if author or title is empty or sanitizes to empty.
func TargetPath(ctx context.Context, filePath, libraryRoot, author, title string) string {
	if author == "" || title == "" {
		slog.DebugContext(ctx, "organize: skipping reorganization due to empty author or title")
		return ""
	}
	safeAuthor := sanitizeDirName(author)
	safeTitle := sanitizeDirName(title)
	if safeAuthor == "" || safeTitle == "" {
		slog.DebugContext(ctx, "organize: skipping reorganization due to sanitized empty author or title")
		return ""
	}
	return filepath.Join(libraryRoot, safeAuthor, safeTitle, filepath.Base(filePath))
}

// sanitizeDirName cleans a string for use as a directory name.
func sanitizeDirName(name string) string {
	name = strings.TrimSpace(name)
	// Remove characters that are problematic in directory names across platforms.
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', '\x00', ':', '*', '?', '"', '<', '>', '|':
			return -1
		default:
			return r
		}
	}, name)
	// Remove leading dots (hidden dirs on Unix).
	name = strings.TrimLeft(name, ".")
	return strings.TrimSpace(name)
}
