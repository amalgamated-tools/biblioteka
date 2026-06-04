package middleware

import (
	"net/url"
	"testing"
)

func TestRedactRequestURL(t *testing.T) {
	t.Parallel()

	raw := "https://example.com/kobo/secret-token/v1/library/sync?x=1"
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	got := redactRequestURL(parsed)
	want := "https://example.com/kobo/%5Bredacted%5D/v1/library/sync?x=1"
	if got != want {
		t.Fatalf("redactRequestURL() = %q, want %q", got, want)
	}
}
