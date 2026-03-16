package pathparser

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
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

// parseAuthorSeriesDirs handles: "Author/Series/[N.] Title [- Author] [(Year)].ext"
// With 2+ directory levels, dirs[0] = Author, dirs[1] = Series.
func parseAuthorSeriesDirs(dirs []string, baseName string) PathInfo {
	info := PathInfo{
		Author:     dirs[0],
		SeriesName: dirs[1],
	}
	info.Title, info.SeriesPosition, info.Year = parseFilenameComponents(baseName)
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

var reSeriesPos = regexp.MustCompile(`^(\d+)\.\s+`)
var reYear = regexp.MustCompile(`\((\d{4})\)\s*$`)
var reTrailingAuthor = regexp.MustCompile(`\s+-\s+[^-]+$`)

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
	// looks like an author name (doesn't contain digits that might be a subtitle).
	loc := reTrailingAuthor.FindStringIndex(name)
	if loc == nil {
		return name
	}
	// Extract the trailing part (excluding the separator) and check for digits.
	suffix := strings.TrimSpace(name[loc[0]:])
	if strings.HasPrefix(suffix, "-") {
		suffix = strings.TrimSpace(strings.TrimPrefix(suffix, "-"))
	}
	if suffix == "" {
		return name
	}
	for _, r := range suffix {
		if unicode.IsDigit(r) {
			// Likely a subtitle such as "Part 1"; don't strip.
			return name
		}
	}
	return strings.TrimSpace(name[:loc[0]])
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
