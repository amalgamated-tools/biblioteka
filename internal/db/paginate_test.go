package db

import (
	"context"
	"strings"
	"testing"
)

func TestListPaginated_RejectsUnsupportedQueryParts(t *testing.T) {
	d := newTestDB(t)

	_, _, err := listPaginated(
		context.Background(),
		d,
		"authors",
		authorColumns,
		"ORDER BY injected",
		10,
		0,
		func(row interface{ Scan(...any) error }) (*Author, error) {
			return scanAuthor(row)
		},
	)
	if err == nil {
		t.Fatal("expected error for unsupported query parts")
	}
	if !strings.Contains(err.Error(), "unsupported query parts") {
		t.Fatalf("unexpected error: %v", err)
	}
}
