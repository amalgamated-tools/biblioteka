package db

import (
	"context"
	"database/sql"
)

// findOrCreate implements the lookup → insert → race-fetch pattern common
// to named-entity tables. normalize, getByName, and create are
// entity-specific callbacks. errInvalid is returned when the normalized name
// is blank; errExists is the sentinel returned by create on a unique-constraint
// violation, which triggers a final getByName to retrieve the row that won
// the race.
func findOrCreate[T any](
	ctx context.Context,
	name string,
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

	t, err := getByName(ctx, name)
	if err == nil {
		return t, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Not found — insert.
	t, err = create(ctx, name)
	if err == nil {
		return t, nil
	}
	if err != errExists {
		return nil, err
	}

	// Concurrent insert won the race — fetch.
	return getByName(ctx, name)
}
