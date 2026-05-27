package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// slogAdapter wraps the global slog logger to satisfy the asynq.Logger
// interface. This routes all asynq-internal log output through the app's
// structured JSON handler and OTEL log exporter instead of asynq's default
// stdout logger.
//
// asynq's Logger interface does not pass a context.Context, so we use
// context.Background() to satisfy sloglint's requirement for *Context variants.
type slogAdapter struct{}

func (slogAdapter) Debug(args ...any) {
	slog.DebugContext(context.Background(), fmt.Sprint(args...))
}

func (slogAdapter) Info(args ...any) {
	slog.InfoContext(context.Background(), fmt.Sprint(args...))
}

func (slogAdapter) Warn(args ...any) {
	slog.WarnContext(context.Background(), fmt.Sprint(args...))
}

func (slogAdapter) Error(args ...any) {
	slog.ErrorContext(context.Background(), fmt.Sprint(args...))
}

// Fatal logs at Error level and then exits with status 1, matching the
// behavior that asynq expects for fatal log entries.
func (slogAdapter) Fatal(args ...any) {
	slog.ErrorContext(context.Background(), fmt.Sprint(args...))
	os.Exit(1)
}
