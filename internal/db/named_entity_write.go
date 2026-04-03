package db

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// namedEntityCreate normalizes name, validates it, executes insertFn, and
// translates unique-constraint violations to errExists. A warn-level log is
// emitted when the normalized name is blank; a debug-level log is emitted on
// a successful insert path.
func namedEntityCreate[T any](
	ctx context.Context,
	entityLabel string,
	name string,
	normalize func(string) string,
	errInvalid, errExists error,
	insertFn func(context.Context, string) (*T, error),
) (*T, error) {
	name = normalize(name)
	if name == "" {
		slog.WarnContext(ctx, "db: rejecting "+entityLabel+" with blank name after normalization")
		return nil, errInvalid
	}
	slog.DebugContext(ctx, "db: creating "+entityLabel, slog.String(otelkeys.Name, name))
	result, err := insertFn(ctx, name)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, errExists
		}
		return nil, err
	}
	return result, nil
}

// namedEntityUpdate normalizes name, validates it, executes updateFn, and
// translates unique-constraint violations to errExists. A warn-level log is
// emitted when the normalized name is blank; a debug-level log is emitted on
// a successful update path.
func namedEntityUpdate[T any](
	ctx context.Context,
	entityLabel string,
	id, name string,
	normalize func(string) string,
	errInvalid, errExists error,
	updateFn func(context.Context, string, string) (*T, error),
) (*T, error) {
	name = normalize(name)
	if name == "" {
		slog.WarnContext(ctx, "db: rejecting "+entityLabel+" update with blank name after normalization")
		return nil, errInvalid
	}
	slog.DebugContext(ctx, "db: updating "+entityLabel,
		slog.String(otelkeys.ID, id),
		slog.String(otelkeys.Name, name),
	)
	result, err := updateFn(ctx, id, name)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, errExists
		}
		return nil, err
	}
	return result, nil
}
