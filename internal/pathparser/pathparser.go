package pathparser

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// PathInfo contains structured metadata extracted from a book file's path
// relative to a library root directory.
type PathInfo struct {
	Author         string
	Title          string
	SeriesName     string
	SeriesPosition *float64
	Year           *int
}

// ParseBookPath extracts author, title, series, and year information from a
// book file's absolute path relative to the library root. It recognizes
// several common directory layouts:
//
//   - 3+ segments: Author/Series/[N.] Title [- Author] [(Year)].ext
//   - 2 segments:  Author/[N.] Title [- Author] [(Year)].ext
//   - 1 segment:   Author - Title/filename.ext
//   - 0 segments:  Author - Title.ext (flat file)
func ParseBookPath(filePath, libraryRoot string) PathInfo {
	relPath, err := filepath.Rel(libraryRoot, filePath)
	if err != nil {
		// Fall back to just filename if Rel fails.
		relPath = filepath.Base(filePath)
	} else {
		// filepath.Rel can return paths starting with ".." when filePath is
		// outside libraryRoot. In that case, or if the result is somehow
		// absolute, avoid treating the leading segments as real author/series
		// directories and instead fall back to just the base filename.
		relSlash := filepath.ToSlash(relPath)
		if filepath.IsAbs(relPath) || relSlash == ".." || strings.HasPrefix(relSlash, "../") {
			relPath = filepath.Base(filePath)
		}
	}

	parts := strings.Split(filepath.ToSlash(relPath), "/")
	filename := parts[len(parts)-1]
	dirs := parts[:len(parts)-1]

	baseName := stripExtension(filename)

	switch len(dirs) {
	case 0:
		return parseFlatFile(baseName)
	case 1:
		// Could be "Author/file.ext" or "Author - Title/file.ext"
		if _, _, found := strings.Cut(dirs[0], " - "); found {
			return parseDashDir(dirs[0], baseName)
		}
		return parseAuthorDir(dirs[0], baseName)
	default:
		// 2+ dirs: Author/Series/file.ext (or deeper)
		return parseAuthorSeriesDirs(dirs, baseName)
	}
}

// parseFlatFile handles: "Author - Title.ext" or "Title - Author.ext"
// Without external context, we assume "Author - Title" order.
func parseFlatFile(baseName string) PathInfo {
	info := PathInfo{}
	author, title := splitDash(baseName)
	if title == "" {
		info.Title = cleanTitle(baseName)
		return info
	}
	info.Author = strings.TrimSpace(author)
	info.Title, info.SeriesPosition, info.Year = parseFilenameComponents(title)
	return info
}

// parseDashDir handles: "Author - Title/filename.ext"
// The directory name contains a " - " separator with author and title.
// The filename often repeats the directory name, so we prefer the directory
// for author/title and only extract series position and year from the filename.
func parseDashDir(dir, baseName string) PathInfo {
	author, dirTitle := splitDash(dir)
	info := PathInfo{
		Author: strings.TrimSpace(author),
		Title:  cleanTitle(dirTitle),
	}
	// Extract supplemental info (year, series position) from filename.
	info.Year = extractYear(baseName)
	info.SeriesPosition = extractSeriesPosition(baseName)
	return info
}

// parseAuthorDir handles: "Author/[N.] Title [- Author] [(Year)].ext"
func parseAuthorDir(authorDir, baseName string) PathInfo {
	info := PathInfo{
		Author: authorDir,
	}
	info.Title, info.SeriesPosition, info.Year = parseFilenameComponents(baseName)
	return info
}

// normalizeName produces a canonical form for comparing author/series/title
// directory names and parsed titles. It lowercases, trims, and removes
// non-letter/digit runes so that minor punctuation or spacing differences
// don't matter when deciding whether two names are effectively the same.
func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func namesEqual(a, b string) bool {
	return normalizeName(a) == normalizeName(b)
}

// parseAuthorSeriesDirs handles: "Author/Series/[N.] Title [- Author] [(Year)].ext"
// With 2+ directory levels, dirs[0] = Author, dirs[1] = Series.
// If the filename does not carry a leading series-position prefix (e.g. "1. "),
// dirs[1] is treated as a title directory (Calibre-style Author/Title/file.ext)
// rather than a series name, to avoid creating phantom series records.
// Additionally, when there is a series-position prefix, dirs[1] is only treated
// as a series name if it does not effectively duplicate the parsed title.
func parseAuthorSeriesDirs(dirs []string, baseName string) PathInfo {
	info := PathInfo{
		Author: dirs[0],
	}
	info.Title, info.SeriesPosition, info.Year = parseFilenameComponents(baseName)

	// Only treat dirs[1] as a series name when the filename has a series
	// position prefix — otherwise it's just a title-level directory. Even when
	// there is a series position, avoid creating a phantom series if dirs[1]
	// is effectively the same as the parsed title (e.g. Author/Title/1. Title.ext).
	if info.SeriesPosition != nil && !namesEqual(dirs[1], info.Title) {
		info.SeriesName = dirs[1]
	}
	return info
}

// parseFilenameComponents extracts title, optional series position, and optional
// year from a filename (without extension). Handles patterns like:
//
//	"10. Tea Time for the Traditionally Built - Alexander McCall Smith (2009)"
//	"1. The Seven Dials Mystery - Agatha Christie (2010)"
//	"Mary Shelley - Frankenstein"
func parseFilenameComponents(name string) (title string, pos *float64, year *int) {
	name = strings.TrimSpace(name)

	// Extract trailing year: "(2009)"
	year = extractYear(name)
	if year != nil {
		name = reYear.ReplaceAllString(name, "")
		name = strings.TrimSpace(name)
	}

	// Strip trailing " - Author" (common in filenames that repeat the author).
	name = stripTrailingAuthor(name)

	// Extract leading series position: "10. " or "1. "
	pos = extractSeriesPosition(name)
	if pos != nil {
		name = reSeriesPos.ReplaceAllString(name, "")
		name = strings.TrimSpace(name)
	}

	title = cleanTitle(name)
	return title, pos, year
}

var (
	reSeriesPos = regexp.MustCompile(`^(\d+)\.\s+`)
	reYear      = regexp.MustCompile(`\((\d{4})\)\s*$`)
)

func extractSeriesPosition(name string) *float64 {
	m := reSeriesPos.FindStringSubmatch(name)
	if m == nil {
		return nil
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return nil
	}
	return &n
}

func extractYear(name string) *int {
	m := reYear.FindStringSubmatch(name)
	if m == nil {
		return nil
	}
	y, err := strconv.Atoi(m[1])
	if err != nil {
		return nil
	}
	return &y
}

func stripTrailingAuthor(name string) string {
	// Only strip if there's a " - " separator and the part after it
	// looks like an author name. Be conservative to avoid truncating
	// legitimate subtitles such as "Title - A Novel" or "Title - Special Edition".
	idx := strings.LastIndex(name, " - ")
	if idx == -1 {
		return name
	}

	suffix := strings.TrimSpace(name[idx+3:])
	if suffix == "" {
		return name
	}

	if !isLikelyPersonName(suffix) {
		// Suffix doesn't look like a personal name; keep the original string.
		return name
	}

	return strings.TrimSpace(name[:idx])
}

// isLikelyPersonName applies a conservative heuristic to decide whether s
// looks like a human author name (e.g. "Jane Doe", "Isaac Asimov").
// It intentionally prefers false negatives (not stripping) over false positives.
func isLikelyPersonName(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	// Quick reject if there are any digits.
	for _, r := range s {
		if unicode.IsDigit(r) {
			return false
		}
	}

	parts := strings.Fields(s)
	if len(parts) < 2 || len(parts) > 4 {
		// We require at least two words (given name + surname) to avoid
		// false positives on single-word subtitle fragments like "Novel",
		// "Unabridged", or "Remastered" that commonly appear after " - ".
		// This means single-surname suffixes like "Shelley" won't be
		// stripped — an acceptable trade-off for avoiding title corruption.
		return false
	}

	// Reject leading articles that are common in subtitles but rare in names.
	switch strings.ToLower(parts[0]) {
	case "a", "an", "the":
		return false
	}

	for _, part := range parts {
		r, _ := utf8.DecodeRuneInString(part)
		if r == utf8.RuneError {
			return false
		}
		if !unicode.IsLetter(r) || !unicode.IsUpper(r) {
			// Names typically start with an uppercase letter.
			return false
		}
	}

	return true
}

func splitDash(s string) (left, right string) {
	before, after, found := strings.Cut(s, " - ")
	if !found {
		return s, ""
	}
	return before, after
}

func stripExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return filename
	}
	return strings.TrimSuffix(filename, ext)
}

func cleanTitle(s string) string {
	return strings.TrimSpace(s)
}
