package organize

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReorganizeFile moves a book file into the canonical Author/Title/ directory
// structure under libraryRoot. If the file is already in the correct location,
// it returns the original path unchanged. After moving, empty source
// directories are cleaned up (up to but not including libraryRoot).
//
// Returns the new absolute path of the file.
func ReorganizeFile(filePath, libraryRoot, author, title string) (string, error) {
	if author == "" || title == "" {
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
		return "", fmt.Errorf("target path %q escapes library root %q", targetPath, libraryRoot)
	}

	// Already in the right place.
	if filepath.Clean(filePath) == filepath.Clean(targetPath) {
		return filePath, nil
	}

	// Create target directory.
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("create target directory %s: %w", targetDir, err)
	}

	// Fail fast if a different file already exists at the target path to avoid
	// silently overwriting data.
	if _, err := os.Stat(targetPath); err == nil {
		return "", fmt.Errorf("target file already exists: %s", targetPath)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat target file %s: %w", targetPath, err)
	}

	// Try rename first (fast, same-filesystem).
	if err := os.Rename(filePath, targetPath); err == nil {
		cleanEmptyDirs(filepath.Dir(filePath), libraryRoot)
		return targetPath, nil
	}

	// Cross-filesystem fallback: copy then remove.
	if err := copyFile(filePath, targetPath); err != nil {
		return "", fmt.Errorf("copy file to %s: %w", targetPath, err)
	}
	if err := os.Remove(filePath); err != nil {
		// File was copied but we couldn't remove the original.
		// Return the new path but surface the cleanup error so callers can log/handle it.
		return targetPath, fmt.Errorf("remove original file %s after copy to %s: %w", filePath, targetPath, err)
	}

	cleanEmptyDirs(filepath.Dir(filePath), libraryRoot)
	return targetPath, nil
}

// cleanEmptyDirs removes empty directories from dir up to (but not including)
// stopAt. Stops as soon as a non-empty directory is encountered.
func cleanEmptyDirs(dir, stopAt string) {
	stopAt = filepath.Clean(stopAt)
	for {
		dir = filepath.Clean(dir)

		// Ensure dir is a strict descendant of stopAt.
		rel, err := filepath.Rel(stopAt, dir)
		if err != nil {
			return
		}
		// If dir is the same as stopAt, or outside/above it, stop.
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
			return
		}

		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Capture source file metadata so we can preserve it on the destination.
	srcInfo, err := in.Stat()
	if err != nil {
		return err
	}
	srcMode := srcInfo.Mode()
	srcModTime := srcInfo.ModTime()

	dstDir := filepath.Dir(dst)
	out, err := os.CreateTemp(dstDir, ".biblioteka-copy-*")
	if err != nil {
		return err
	}
	tmpName := out.Name()

	// Ensure the temp file has the same permissions as the source file, so that
	// the copy behaves like a rename from the user's perspective.
	if err := out.Chmod(srcMode.Perm()); err != nil {
		_ = out.Close()
		return err
	}

	defer func() {
		// Ensure the temp file is cleaned up on any error.
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	// Close the temp file before renaming to ensure contents are flushed and to
	// avoid rename failures on platforms like Windows.
	if cerr := out.Close(); cerr != nil {
		return cerr
	}

	// Preserve modification time (and use it for access time as well) so the
	// copied file closely matches the original's metadata.
	if err = os.Chtimes(tmpName, srcModTime, srcModTime); err != nil {
		return err
	}

	// Fail fast if the destination already exists instead of overwriting it.
	if _, statErr := os.Stat(dst); statErr == nil {
		return fmt.Errorf("destination file %q already exists", dst)
	} else if !os.IsNotExist(statErr) {
		return statErr
	}

	if err = os.Rename(tmpName, dst); err != nil {
		return err
	}

	return nil
}

// ReorganizeFileFlat moves a book file into an Author/ directory structure
// under libraryRoot (flat, no title subdirectory). If the file is already in
// the correct location, it returns the original path unchanged. After moving,
// empty source directories are cleaned up (up to but not including libraryRoot).
//
// Returns the new absolute path of the file.
func ReorganizeFileFlat(filePath, libraryRoot, author string) (string, error) {
	if author == "" {
		return filePath, nil
	}

	safeAuthor := sanitizeDirName(author)
	if safeAuthor == "" {
		return filePath, nil
	}

	filename := filepath.Base(filePath)
	targetDir := filepath.Join(libraryRoot, safeAuthor)
	targetPath := filepath.Join(targetDir, filename)

	// Defense-in-depth: verify the target path is still inside libraryRoot.
	relCheck, err := filepath.Rel(libraryRoot, targetPath)
	if err != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("target path %q escapes library root %q", targetPath, libraryRoot)
	}

	// Already in the right place.
	if filepath.Clean(filePath) == filepath.Clean(targetPath) {
		return filePath, nil
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("create target directory %s: %w", targetDir, err)
	}

	if _, err := os.Stat(targetPath); err == nil {
		return "", fmt.Errorf("target file already exists: %s", targetPath)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat target file %s: %w", targetPath, err)
	}

	if err := os.Rename(filePath, targetPath); err == nil {
		cleanEmptyDirs(filepath.Dir(filePath), libraryRoot)
		return targetPath, nil
	}

	// Cross-filesystem fallback: copy then remove.
	if err := copyFile(filePath, targetPath); err != nil {
		return "", fmt.Errorf("copy file to %s: %w", targetPath, err)
	}
	if err := os.Remove(filePath); err != nil {
		return targetPath, fmt.Errorf("remove original file %s after copy to %s: %w", filePath, targetPath, err)
	}

	cleanEmptyDirs(filepath.Dir(filePath), libraryRoot)
	return targetPath, nil
}

// TargetPathFlat returns the canonical target file path that
// ReorganizeFileFlat would produce for the given inputs, without actually
// moving anything. Returns an empty string if author is empty or sanitizes
// to empty.
func TargetPathFlat(filePath, libraryRoot, author string) string {
	if author == "" {
		return ""
	}
	safeAuthor := sanitizeDirName(author)
	if safeAuthor == "" {
		return ""
	}
	return filepath.Join(libraryRoot, safeAuthor, filepath.Base(filePath))
}

// TargetPath returns the canonical target file path that ReorganizeFile would
// produce for the given inputs, without actually moving anything. Returns an
// empty string if author or title is empty or sanitizes to empty.
func TargetPath(filePath, libraryRoot, author, title string) string {
	if author == "" || title == "" {
		return ""
	}
	safeAuthor := sanitizeDirName(author)
	safeTitle := sanitizeDirName(title)
	if safeAuthor == "" || safeTitle == "" {
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
