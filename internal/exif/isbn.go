package exif

import "strings"

// isASIN reports whether every byte of s is an alphanumeric character (digit or ASCII letter).
// The caller is responsible for enforcing that len(s) == 10 before treating the value as an ASIN.
func isASIN(s string) bool {
	for i := range s {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			return false
		}
	}
	return true
}

// NormalizeISBN strips common prefixes (urn:isbn:, isbn:), leading/trailing
// whitespace, hyphens, and internal spaces from a raw ISBN string. It returns
// the cleaned value only if it looks like an ISBN-10 or ISBN-13: 10 or 13
// characters consisting of digits, with ISBN-10 allowing an 'X' (or 'x') as
// the final checksum character; otherwise it returns "".
func NormalizeISBN(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "urn:isbn:"):
		s = s[len("urn:isbn:"):]
	case strings.HasPrefix(lower, "isbn:"):
		s = s[len("isbn:"):]
	}
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimSpace(s)

	switch len(s) {
	case 10:
		for i := range 9 {
			if s[i] < '0' || s[i] > '9' {
				return ""
			}
		}
		last := s[9]
		if (last < '0' || last > '9') && last != 'X' && last != 'x' {
			return ""
		}
		if last == 'x' {
			s = s[:9] + "X"
		}
		return s
	case 13:
		for i := range 13 {
			if s[i] < '0' || s[i] > '9' {
				return ""
			}
		}
		return s
	default:
		return ""
	}
}
