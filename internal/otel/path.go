package otel

import "strings"

// SanitizePathForTelemetry redacts sensitive path segments before they are used
// in logs and span names.
func SanitizePathForTelemetry(path string) string {
	if !strings.HasPrefix(path, "/kobo/") {
		return path
	}

	rest := strings.TrimPrefix(path, "/kobo/")
	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		return "/kobo/REDACTED"
	}

	return "/kobo/REDACTED" + rest[slashIdx:]
}
