package db

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// namedEntityCreate normalizes name, validates it, executes insertFn, and
// translates unique-constraint violations to errExists. A warn-level log is
// emitted when the normalized name is blank; a debug-level log is emitted
// before the insert is attempted.
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
		slog.WarnContext(ctx, "rejecting entity with blank name after normalization", slog.String(otelkeys.EntityType, entityLabel))
		return nil, errInvalid
	}
	slog.DebugContext(ctx, "db: creating entity",
		slog.String(otelkeys.EntityType, entityLabel),
		slog.String(otelkeys.Name, name),
	)
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
// emitted when the normalized name is blank; a debug-level log is emitted
// before the update is attempted.
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
		slog.WarnContext(ctx, "rejecting entity update with blank name after normalization", slog.String(otelkeys.EntityType, entityLabel))
		return nil, errInvalid
	}
	slog.DebugContext(ctx, "db: updating entity",
		slog.String(otelkeys.EntityType, entityLabel),
		slog.String(otelkeys.EntityID, id),
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
