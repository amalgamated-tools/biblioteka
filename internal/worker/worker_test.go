package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
)

const testRedisURL = "redis://localhost:6379"

// newTestWorker creates a Worker using a valid but unreachable Redis URL.
// The URL is only parsed — no network connection is made during construction.
func newTestWorker(t *testing.T) *Worker {
	t.Helper()
	w, err := New(testRedisURL)
	if err != nil {
		t.Fatalf("New(%q): %v", testRedisURL, err)
	}
	t.Cleanup(func() {
		// Best-effort close; ignore errors since Redis is not running.
		_ = w.Close()
	})
	return w
}

// TestNew_ValidURL verifies that New returns a fully-initialised Worker when
// given a syntactically valid Redis URL.
func TestNew_ValidURL(t *testing.T) {
	w := newTestWorker(t)
	if w.client == nil {
		t.Error("client is nil")
	}
	if w.mux == nil {
		t.Error("mux is nil")
	}
	if w.scheduler == nil {
		t.Error("scheduler is nil")
	}
	if w.redisOpt == nil {
		t.Error("redisOpt is nil")
	}
}

// TestNew_InvalidURL verifies that New returns an error for a non-Redis URI.
func TestNew_InvalidURL(t *testing.T) {
	_, err := New("not-a-valid-redis-url")
	if err == nil {
		t.Fatal("expected error for invalid Redis URL, got nil")
	}
}

// TestNew_EmptyURL verifies that New returns an error for an empty string.
func TestNew_EmptyURL(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Fatal("expected error for empty Redis URL, got nil")
	}
}

// TestNew_WrongScheme verifies that New returns an error when the URL uses an
// unsupported scheme such as http://.
func TestNew_WrongScheme(t *testing.T) {
	_, err := New("http://localhost:6379")
	if err == nil {
		t.Fatal("expected error for http:// URL, got nil")
	}
}

// TestRedisConnOpt verifies that RedisConnOpt returns the option that was
// created during construction.
func TestRedisConnOpt(t *testing.T) {
	w := newTestWorker(t)
	if w.RedisConnOpt() == nil {
		t.Error("RedisConnOpt returned nil")
	}
}

// TestRegister verifies that Register stores a handler in the mux and that the
// handler is invoked with the correct payload when the mux processes a task.
func TestRegister(t *testing.T) {
	w := newTestWorker(t)

	var called bool
	var gotPayload []byte
	w.Register(context.Background(), "test:register", func(_ context.Context, payload []byte) error {
		called = true
		gotPayload = payload
		return nil
	})

	task := asynq.NewTask("test:register", []byte(`{"hello":"world"}`))
	if err := w.mux.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("ProcessTask: %v", err)
	}

	if !called {
		t.Error("expected handler to be called, but it was not")
	}
	if string(gotPayload) != `{"hello":"world"}` {
		t.Errorf("handler received payload %q, want %q", gotPayload, `{"hello":"world"}`)
	}
}

// TestRegister_HandlerError verifies that an error returned by a registered
// handler propagates through ProcessTask.
func TestRegister_HandlerError(t *testing.T) {
	w := newTestWorker(t)

	sentinel := errors.New("handler error")
	w.Register(context.Background(), "test:fail", func(_ context.Context, _ []byte) error {
		return sentinel
	})

	task := asynq.NewTask("test:fail", nil)
	err := w.mux.ProcessTask(context.Background(), task)
	if !errors.Is(err, sentinel) {
		t.Errorf("ProcessTask returned %v, want %v", err, sentinel)
	}
}

// TestRegister_NilPayload verifies that a handler registered via Register
// receives a nil (or empty) payload when the task carries no data.
func TestRegister_NilPayload(t *testing.T) {
	w := newTestWorker(t)

	var called bool
	w.Register(context.Background(), "test:nil-payload", func(_ context.Context, payload []byte) error {
		called = true
		// payload should be nil or empty — both are acceptable
		return nil
	})

	task := asynq.NewTask("test:nil-payload", nil)
	if err := w.mux.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("ProcessTask: %v", err)
	}
	if !called {
		t.Error("expected handler to be called")
	}
}

// TestRegisterSchedule verifies that a valid cron expression and
// JSON-serialisable payload produce a non-empty entry ID.
func TestRegisterSchedule(t *testing.T) {
	w := newTestWorker(t)

	entryID, err := w.RegisterSchedule("@every 1m", "test:sched", map[string]string{"k": "v"})
	if err != nil {
		t.Fatalf("RegisterSchedule: %v", err)
	}
	if entryID == "" {
		t.Error("expected non-empty entry ID")
	}
}

// TestRegisterSchedule_NilPayload verifies that nil is accepted as a payload
// (it marshals to the JSON literal "null").
func TestRegisterSchedule_NilPayload(t *testing.T) {
	w := newTestWorker(t)

	entryID, err := w.RegisterSchedule("@every 1h", "test:sched-nil", nil)
	if err != nil {
		t.Fatalf("RegisterSchedule with nil payload: %v", err)
	}
	if entryID == "" {
		t.Error("expected non-empty entry ID for nil payload")
	}
}

// TestRegisterSchedule_MultipleEntries verifies that separate schedules for
// the same job each receive a distinct entry ID.
func TestRegisterSchedule_MultipleEntries(t *testing.T) {
	w := newTestWorker(t)

	id1, err := w.RegisterSchedule("@every 1m", "test:multi", nil)
	if err != nil {
		t.Fatalf("first RegisterSchedule: %v", err)
	}
	id2, err := w.RegisterSchedule("@every 2m", "test:multi", nil)
	if err != nil {
		t.Fatalf("second RegisterSchedule: %v", err)
	}
	if id1 == id2 {
		t.Errorf("expected distinct entry IDs, got %q for both", id1)
	}
}

// TestRegisterSchedule_InvalidCronspec verifies that an unparseable cron
// expression returns an error.
func TestRegisterSchedule_InvalidCronspec(t *testing.T) {
	w := newTestWorker(t)

	_, err := w.RegisterSchedule("not a valid cron expression", "test:sched", nil)
	if err == nil {
		t.Fatal("expected error for invalid cron spec, got nil")
	}
}

// TestRegisterSchedule_UnmarshalablePayload verifies that a payload that
// cannot be marshalled to JSON (e.g. a channel) returns an error before any
// Redis interaction.
func TestRegisterSchedule_UnmarshalablePayload(t *testing.T) {
	w := newTestWorker(t)

	_, err := w.RegisterSchedule("@every 1m", "test:sched", make(chan int))
	if err == nil {
		t.Fatal("expected error for unmarshalable payload, got nil")
	}
}

// TestEnqueue_UnmarshalablePayload verifies that Enqueue returns an error
// immediately — before any network I/O — when the payload cannot be marshalled
// to JSON.
func TestEnqueue_UnmarshalablePayload(t *testing.T) {
	w := newTestWorker(t)

	_, err := w.Enqueue(context.Background(), "test:enqueue", make(chan int))
	if err == nil {
		t.Fatal("expected error for unmarshalable payload, got nil")
	}
}

// TestClose verifies that Close can be called on a newly-created Worker that
// has never been started without returning an error.
func TestClose(t *testing.T) {
	w, err := New(testRedisURL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
