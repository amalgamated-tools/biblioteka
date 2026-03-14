package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

const (
	// QueueName is the name of the default job queue.
	QueueName = "default"

	// DefaultConcurrency is the max number of concurrent jobs.
	DefaultConcurrency = 4

	// DefaultMaxRetry is how many times a failed job is retried.
	DefaultMaxRetry = 5
)

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

	return &Worker{
		client:    asynq.NewClient(opt),
		mux:       asynq.NewServeMux(),
		scheduler: asynq.NewScheduler(opt, nil),
		redisOpt:  opt,
	}, nil
}

// Register a named job handler. Must be called before Start.
func (w *Worker) Register(name string, fn Func) {
	slog.Debug("registering job handler", slog.String("job", name))
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
	task := asynq.NewTask(jobName, body, asynq.MaxRetry(DefaultMaxRetry), asynq.Queue(QueueName))
	entryID, err := w.scheduler.Register(cronspec, task)
	if err != nil {
		return "", fmt.Errorf("register schedule %s %q: %w", jobName, cronspec, err)
	}
	return entryID, nil
}

// Start begins processing jobs and running scheduled tasks, blocking until ctx
// is cancelled.
func (w *Worker) Start(ctx context.Context) {
	slog.DebugContext(ctx, "starting asynq worker", slog.Int("concurrency", DefaultConcurrency))
	srv := asynq.NewServer(w.redisOpt, asynq.Config{
		Concurrency: DefaultConcurrency,
		Queues:      map[string]int{QueueName: 1},
	})

	if err := srv.Start(w.mux); err != nil {
		slog.ErrorContext(ctx, "Failed to start asynq server", slog.Any("error", err))
		return
	}

	if err := w.scheduler.Start(); err != nil {
		slog.Error("Failed to start asynq scheduler", slog.Any("error", err))
		srv.Shutdown()
		return
	}

	<-ctx.Done()
	w.scheduler.Shutdown()
	srv.Shutdown()
}

// Enqueue adds a job to the queue with the given name and JSON-serialisable payload.
func (w *Worker) Enqueue(ctx context.Context, name string, payload any) (string, error) {
	slog.DebugContext(ctx, "enqueuing job", slog.String("job", name))
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	task := asynq.NewTask(name, body, asynq.MaxRetry(DefaultMaxRetry), asynq.Queue(QueueName))
	info, err := w.client.EnqueueContext(ctx, task)
	if err != nil {
		return "", fmt.Errorf("enqueue task %s: %w", name, err)
	}
	slog.DebugContext(ctx, "job enqueued", slog.String("job", name), slog.String("task_id", info.ID))
	return info.ID, nil
}

// RedisConnOpt returns the Redis connection option used by this worker.
func (w *Worker) RedisConnOpt() asynq.RedisConnOpt {
	return w.redisOpt
}

// Close shuts down the asynq client connection.
func (w *Worker) Close() error {
	return w.client.Close()
}
