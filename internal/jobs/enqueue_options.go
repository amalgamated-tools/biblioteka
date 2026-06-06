package jobs

import "time"

// EnqueueOption is a functional option for Worker.Enqueue.
// Options are applied in order; later options override earlier ones.
type EnqueueOption func(*EnqueueOptions)

// EnqueueOptions holds the resolved configuration for a single Enqueue call.
// It is exported so that the worker package can read the applied values.
type EnqueueOptions struct {
	Unique   time.Duration // 0 means no uniqueness constraint
	MaxRetry int           // 0 means use the worker default
	Queue    string        // "" means use the worker default queue
}

// ApplyEnqueueOptions returns an EnqueueOptions struct with all provided
// options applied in order. It is exported for use by the worker package.
func ApplyEnqueueOptions(opts []EnqueueOption) EnqueueOptions {
	var o EnqueueOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithUnique sets a deduplication window: if the same task type and payload
// are enqueued again within d, the enqueue is rejected and the underlying
// asynq client returns an error that wraps asynq.ErrDuplicateTask (use
// errors.Is to detect it). Pass 0 to disable uniqueness (the default).
func WithUnique(d time.Duration) EnqueueOption {
	return func(o *EnqueueOptions) { o.Unique = d }
}
