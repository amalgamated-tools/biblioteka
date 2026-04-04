package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"

	"github.com/stretchr/testify/require"
)

const testRedisURL = "redis://192.0.2.1:6379" // TEST-NET (RFC 5737) — guaranteed non-routable

// newTestWorker creates a Worker using a valid but unreachable Redis URL.
// The URL is only parsed — no network connection is made during construction.
func newTestWorker(t *testing.T) *Worker {
	t.Helper()
	w, err := New(testRedisURL)
	require.NoError(t, err, "New(%q)", testRedisURL)
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Logf("cleanup Close: %v", err)
		}
	})
	return w
}

// TestNew_ValidURL verifies that New returns a fully-initialized Worker when
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
	require.Error(t, err, "expected error for invalid Redis URL, got nil")
}

// TestNew_EmptyURL verifies that New returns an error for an empty string.
func TestNew_EmptyURL(t *testing.T) {
	_, err := New("")
	require.Error(t, err, "expected error for empty Redis URL, got nil")
}

// TestNew_WrongScheme verifies that New returns an error when the URL uses an
// unsupported scheme such as http://.
func TestNew_WrongScheme(t *testing.T) {
	_, err := New("http://localhost:6379")
	require.Error(t, err, "expected error for http:// URL, got nil")
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
	// ctx here is only used for the registration-time debug log.
	// The handler receives the context from ProcessTask, not this one.
	w.Register(t.Context(), "test:register", func(_ context.Context, payload []byte) error {
		called = true
		gotPayload = payload
		return nil
	})

	task := asynq.NewTask("test:register", []byte(`{"hello":"world"}`))
	if err := w.mux.ProcessTask(t.Context(), task); err != nil {
		require.NoError(t, err, "ProcessTask")
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
	w.Register(t.Context(), "test:fail", func(_ context.Context, _ []byte) error {
		return sentinel
	})

	task := asynq.NewTask("test:fail", nil)
	err := w.mux.ProcessTask(t.Context(), task)
	if !errors.Is(err, sentinel) {
		t.Errorf("ProcessTask returned %v, want %v", err, sentinel)
	}
}

// TestRegister_NilPayload verifies that a handler registered via Register
// receives a nil (or empty) payload when the task carries no data.
func TestRegister_NilPayload(t *testing.T) {
	w := newTestWorker(t)

	var called bool
	w.Register(t.Context(), "test:nil-payload", func(_ context.Context, payload []byte) error {
		called = true
		if payload != nil && len(payload) != 0 {
			t.Errorf("handler received payload %q, want nil or empty", payload)
		}
		return nil
	})

	task := asynq.NewTask("test:nil-payload", nil)
	if err := w.mux.ProcessTask(t.Context(), task); err != nil {
		require.NoError(t, err, "ProcessTask")
	}
	if !called {
		t.Error("expected handler to be called")
	}
}

// TestRegisterSchedule verifies that a valid cron expression and
// JSON-serializable payload produce a non-empty entry ID.
func TestRegisterSchedule(t *testing.T) {
	w := newTestWorker(t)

	entryID, err := w.RegisterSchedule("@every 1m", "test:sched", map[string]string{"k": "v"})
	require.NoError(t, err, "RegisterSchedule")
	if entryID == "" {
		t.Error("expected non-empty entry ID")
	}
}

// TestRegisterSchedule_NilPayload verifies that nil is accepted as a payload
// (it marshals to the JSON literal "null").
func TestRegisterSchedule_NilPayload(t *testing.T) {
	w := newTestWorker(t)

	entryID, err := w.RegisterSchedule("@every 1h", "test:sched-nil", nil)
	require.NoError(t, err, "RegisterSchedule with nil payload")
	if entryID == "" {
		t.Error("expected non-empty entry ID for nil payload")
	}
}

// TestRegisterSchedule_DistinctIDs verifies that separate schedules for
// the same job each receive a distinct entry ID and that both handlers are
// wired up in the mux.
func TestRegisterSchedule_DistinctIDs(t *testing.T) {
	w := newTestWorker(t)

	var called1, called2 bool
	w.Register(t.Context(), "test:multi-a", func(_ context.Context, _ []byte) error {
		called1 = true
		return nil
	})
	w.Register(t.Context(), "test:multi-b", func(_ context.Context, _ []byte) error {
		called2 = true
		return nil
	})

	id1, err := w.RegisterSchedule("@every 1m", "test:multi-a", nil)
	require.NoError(t, err, "first RegisterSchedule")
	id2, err := w.RegisterSchedule("@every 2m", "test:multi-b", nil)
	require.NoError(t, err, "second RegisterSchedule")
	if id1 == id2 {
		t.Errorf("expected distinct entry IDs, got %q for both", id1)
	}

	// Verify both handlers are actually wired up.
	if err := w.mux.ProcessTask(t.Context(), asynq.NewTask("test:multi-a", nil)); err != nil {
		require.NoError(t, err, "ProcessTask for multi-a")
	}
	if err := w.mux.ProcessTask(t.Context(), asynq.NewTask("test:multi-b", nil)); err != nil {
		require.NoError(t, err, "ProcessTask for multi-b")
	}
	if !called1 {
		t.Error("handler for test:multi-a was not called")
	}
	if !called2 {
		t.Error("handler for test:multi-b was not called")
	}
}

// TestRegisterSchedule_InvalidCronspec verifies that an unparseable cron
// expression returns an error.
func TestRegisterSchedule_InvalidCronspec(t *testing.T) {
	w := newTestWorker(t)

	_, err := w.RegisterSchedule("not a valid cron expression", "test:sched", nil)
	require.Error(t, err, "expected error for invalid cron spec, got nil")
}

// TestRegisterSchedule_NonMarshalablePayload verifies that a payload that
// cannot be marshaled to JSON (e.g. a channel) returns an error before any
// Redis interaction.
func TestRegisterSchedule_NonMarshalablePayload(t *testing.T) {
	w := newTestWorker(t)

	_, err := w.RegisterSchedule("@every 1m", "test:sched", make(chan int))
	require.Error(t, err, "expected error for non-marshalable payload, got nil")
}

// TestEnqueue_NonMarshalablePayload verifies that Enqueue returns an error
// immediately — before any network I/O — when the payload cannot be marshaled
// to JSON.
func TestEnqueue_NonMarshalablePayload(t *testing.T) {
	w := newTestWorker(t)

	_, err := w.Enqueue(t.Context(), "test:enqueue", make(chan int))
	require.Error(t, err, "expected error for non-marshalable payload, got nil")
}

// TestClose verifies that Close can be called on a newly-created Worker that
// has never been started without panicking.
func TestClose(t *testing.T) {
	w, err := New(testRedisURL)
	require.NoError(t, err, "New")
	if err := w.Close(); err != nil {
		// asynq.Client.Close() may error when Redis is unreachable.
		// Accept in unit-test environments; live-Redis path belongs in integration tests.
		t.Logf("Close returned (may be expected without Redis): %v", err)
	}
}
