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
	client   *asynq.Client
	mux      *asynq.ServeMux
	redisOpt asynq.RedisConnOpt
}

// New creates a Worker that connects to Redis at the given URL.
func New(redisURL string) (*Worker, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_URL %q: %w", redisURL, err)
	}

	return &Worker{
		client:   asynq.NewClient(opt),
		mux:      asynq.NewServeMux(),
		redisOpt: opt,
	}, nil
}

// Register a named job handler. Must be called before Start.
func (w *Worker) Register(name string, fn Func) {
	w.mux.HandleFunc(name, func(ctx context.Context, task *asynq.Task) error {
		return fn(ctx, task.Payload())
	})
}

// Start begins processing jobs, blocking until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) {
	srv := asynq.NewServer(w.redisOpt, asynq.Config{
		Concurrency: DefaultConcurrency,
		Queues:      map[string]int{QueueName: 1},
	})

	if err := srv.Start(w.mux); err != nil {
		slog.Error("Failed to start asynq server", slog.Any("error", err))
		return
	}

	<-ctx.Done()
	srv.Shutdown()
}

// Enqueue adds a job to the queue with the given name and JSON-serialisable payload.
func (w *Worker) Enqueue(ctx context.Context, name string, payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	task := asynq.NewTask(name, body, asynq.MaxRetry(DefaultMaxRetry), asynq.Queue(QueueName))
	info, err := w.client.EnqueueContext(ctx, task)
	if err != nil {
		return "", fmt.Errorf("enqueue task %s: %w", name, err)
	}
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
