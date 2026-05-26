// Package worker manages Redis-backed background job processing via asynq.
// It exposes a Worker type that wraps an asynq server, client, and scheduler,
// and provides helpers for registering one-off job handlers and recurring
// cron-scheduled tasks.
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/amalgamated-tools/biblioteka/internal/handlers/middleware"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/hibiken/asynq"
)

const (
	// QueueName is the default priority queue for general jobs.
	// Scheduled periodic jobs use QueueLow instead.
	QueueName = "default"

	// QueueCritical is the high-priority queue for user-initiated actions.
	QueueCritical = "critical"

	// QueueLow is the low-priority queue for background maintenance tasks.
	QueueLow = "low"

	// DefaultConcurrency is the max number of concurrent jobs.
	DefaultConcurrency = 4

	// DefaultMaxRetry is how many times a failed job is retried.
	DefaultMaxRetry = 5

	// DefaultShutdownTimeout is the maximum time to wait for in-flight tasks
	// to finish when the server is asked to stop.
	DefaultShutdownTimeout = 8 * time.Second
)

// taskHeaderCarrier adapts a map[string]string to the propagation.TextMapCarrier
// interface so that the global OTel text-map propagator can inject / extract
// trace context into / from asynq task headers.
type taskHeaderCarrier map[string]string

func (c taskHeaderCarrier) Get(key string) string      { return c[key] }
func (c taskHeaderCarrier) Set(key, value string)       { c[key] = value }
func (c taskHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// Func is the handler function signature for background jobs.
type Func func(ctx context.Context, payload []byte) error

// Worker manages background job processing backed by Redis via asynq.
type Worker struct {
	client    *asynq.Client
	mux       *asynq.ServeMux
	scheduler *asynq.Scheduler
	redisOpt  asynq.RedisConnOpt
}

// New creates a Worker that connects to Redis at the given URL.
func New(redisURL string) (*Worker, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_URL %q: %w", redisURL, err)
	}

	mux := asynq.NewServeMux()
	// Middleware: restore OTel trace context and request ID from task headers so
	// that structured logs and spans emitted inside handlers are correlated with
	// the originating HTTP request.
	mux.Use(traceAndRequestIDMiddleware)
	// Middleware: log a warning when a task arrives with no registered handler.
	mux.Use(notFoundLoggingMiddleware)

	return &Worker{
		client:    asynq.NewClient(opt),
		mux:       mux,
		scheduler: asynq.NewScheduler(opt, nil),
		redisOpt:  opt,
	}, nil
}

// traceAndRequestIDMiddleware extracts the OTel trace context and request ID
// that were stored as task headers at enqueue time and re-injects them into
// the handler context.
func traceAndRequestIDMiddleware(next asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		headers := task.Headers()
		if len(headers) > 0 {
			// Restore OTel trace context (traceparent, tracestate, …).
			ctx = otel.GetTextMapPropagator().Extract(ctx, taskHeaderCarrier(headers))
			// Restore request ID so correlation survives the queue boundary.
			if reqID := headers[otelkeys.RequestID]; reqID != "" {
				ctx = middleware.WithRequestID(ctx, reqID)
				ctx = context.WithValue(ctx, slogRequestIDKey{}, reqID)
			}
		}
		return next.ProcessTask(ctx, task)
	})
}

// slogRequestIDKey is the context key used to attach the request ID to
// slog log records emitted inside job handlers.
type slogRequestIDKey struct{}

// notFoundLoggingMiddleware logs a warning when no handler is registered for
// the incoming task type. It uses the ErrHandlerNotFound sentinel introduced
// in asynq v0.26.0 to detect the case.
func notFoundLoggingMiddleware(next asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		err := next.ProcessTask(ctx, task)
		if err != nil && errors.Is(err, asynq.ErrHandlerNotFound) {
			slog.WarnContext(ctx, "no handler registered for job type",
				slog.String(otelkeys.Job, task.Type()),
			)
		}
		return err
	})
}

// Register a named job handler. Must be called before Start.
func (w *Worker) Register(ctx context.Context, name string, fn Func) {
	slog.DebugContext(ctx, "registering job handler", slog.String(otelkeys.Job, name))
	w.mux.HandleFunc(name, func(ctx context.Context, task *asynq.Task) error {
		return fn(ctx, task.Payload())
	})
}

// RegisterSchedule schedules a job to run repeatedly according to the given
// cron specification (e.g. "@every 1h" or "0 * * * *"). It returns an entry
// ID that can be used to unregister the schedule later. Must be called before
// Start.
func (w *Worker) RegisterSchedule(cronspec, jobName string, payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload for %s: %w", jobName, err)
	}
	task := asynq.NewTask(jobName, body, asynq.MaxRetry(DefaultMaxRetry), asynq.Queue(QueueLow))
	entryID, err := w.scheduler.Register(cronspec, task)
	if err != nil {
		return "", fmt.Errorf("register schedule %s %q: %w", jobName, cronspec, err)
	}
	return entryID, nil
}

// Start begins processing jobs and running scheduled tasks, blocking until ctx
// is cancelled.
func (w *Worker) Start(ctx context.Context) error {
	slog.DebugContext(ctx, "starting asynq worker", slog.Int(otelkeys.Concurrency, DefaultConcurrency))
	srv := asynq.NewServer(w.redisOpt, asynq.Config{
		Concurrency: DefaultConcurrency,
		// Weighted priority queues: user-initiated tasks finish before
		// background maintenance work.
		Queues: map[string]int{
			QueueCritical: 6,
			QueueName:     3,
			QueueLow:      1,
		},
		// Route all asynq-internal log output through the app's slog handler.
		Logger: slogAdapter{},
		// Propagate the root context (tracer, logger) into every job handler.
		BaseContext: func() context.Context { return ctx },
		// Wait up to DefaultShutdownTimeout for in-flight tasks to finish.
		ShutdownTimeout: DefaultShutdownTimeout,
		// Emit a structured warning when Redis becomes temporarily unreachable.
		HealthCheckFunc: func(err error) {
			if err != nil {
				slog.WarnContext(ctx, "Redis health check failed",
					slog.Any(otelkeys.Error, err),
				)
			}
		},
	})

	if err := srv.Start(w.mux); err != nil {
		return fmt.Errorf("start asynq server: %w", err)
	}

	if err := w.scheduler.Start(); err != nil {
		srv.Shutdown()
		return fmt.Errorf("start asynq scheduler: %w", err)
	}

	<-ctx.Done()
	// Stop dequeueing new tasks first so in-flight work can finish cleanly.
	srv.Stop()
	w.scheduler.Shutdown()
	srv.Shutdown()
	return nil
}

// Enqueue adds a job to the queue with the given name and JSON-serialisable
// payload. Options control deduplication, retry count, and target queue.
// Trace context and request ID from ctx are injected into the task headers so
// that end-to-end observability spans survive the queue boundary.
func (w *Worker) Enqueue(ctx context.Context, name string, payload any, opts ...jobs.EnqueueOption) (string, error) {
	slog.DebugContext(ctx, "enqueuing job", slog.String(otelkeys.Job, name))
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	o := jobs.ApplyEnqueueOptions(opts)

	// Build task options.
	maxRetry := DefaultMaxRetry
	if o.MaxRetry > 0 {
		maxRetry = o.MaxRetry
	}
	queue := QueueName
	if o.Queue != "" {
		queue = o.Queue
	}
	asynqOpts := []asynq.Option{
		asynq.MaxRetry(maxRetry),
		asynq.Queue(queue),
	}
	if o.Unique > 0 {
		asynqOpts = append(asynqOpts, asynq.Unique(o.Unique))
	}

	// Propagate trace context and request ID into task headers.
	headers := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, taskHeaderCarrier(headers))
	if reqID := middleware.GetRequestID(ctx); reqID != "" {
		headers[otelkeys.RequestID] = reqID
	}

	task := asynq.NewTaskWithHeaders(name, body, headers, asynqOpts...)
	info, err := w.client.EnqueueContext(ctx, task)
	if err != nil {
		return "", fmt.Errorf("enqueue task %s: %w", name, err)
	}
	slog.DebugContext(ctx, "job enqueued",
		slog.String(otelkeys.Job, name),
		slog.String(otelkeys.TaskID, info.ID),
	)
	return info.ID, nil
}

// RedisConnOpt returns the Redis connection option used by this worker.
func (w *Worker) RedisConnOpt() asynq.RedisConnOpt {
	return w.redisOpt
}

// Close shuts down the scheduler and the asynq client connection.
func (w *Worker) Close() error {
	w.scheduler.Shutdown()
	return w.client.Close()
}

