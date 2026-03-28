package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// findOrCreate implements the lookup → insert → race-fetch pattern common
// to named-entity tables. normalize, getByName, and create are
// entity-specific callbacks. errInvalid is returned when the normalized name
// is blank; errExists is the sentinel returned by create on a unique-constraint
// violation, which triggers a final getByName to retrieve the row that won
// the race.
//
// Normalization and blank-name validation are owned entirely by this function.
// Callers should pass the raw (un-normalized) name. The provided callbacks
// (getByName, create) may re-normalize internally — that is harmless because
// normalization is idempotent — but they are not required to.
func findOrCreate[T any](
	ctx context.Context,
	name string,
	entityLabel string,
	normalize func(string) string,
	errInvalid error,
	errExists error,
	getByName func(context.Context, string) (*T, error),
	create func(context.Context, string) (*T, error),
) (*T, error) {
	name = normalize(name)
	if name == "" {
		return nil, errInvalid
	}

	slog.DebugContext(ctx, "db: find or create "+entityLabel, slog.String(otelkeys.Name, name))

	t, err := getByName(ctx, name)
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Not found — insert.
	t, err = create(ctx, name)
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, errExists) {
		return nil, err
	}

	// Concurrent insert won the race — fetch.
	return getByName(ctx, name)
}
