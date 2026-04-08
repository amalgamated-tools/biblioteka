package pubsub

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMetadataChannel(t *testing.T) {
	ch := MetadataChannel("book-123", "user-456")
	require.Equal(t, "biblioteka:metadata:book-123:user-456", ch)
}

// TestPublishSubscribe is an integration test that requires a running Redis
// instance. It is skipped when REDIS_URL is not set.
func TestPublishSubscribe(t *testing.T) {
	redisURL := redisTestURL(t)

	client, err := NewClient(redisURL)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	channel := "test:pubsub:" + t.Name()
	msgs, unsub := client.Subscribe(ctx, channel)
	defer unsub()

	// Give subscription time to establish.
	time.Sleep(100 * time.Millisecond)

	require.NoError(t, client.Publish(ctx, channel, "hello"))
	require.NoError(t, client.Publish(ctx, channel, "world"))

	got := make([]string, 0, 2)
	for range 2 {
		select {
		case m := <-msgs:
			got = append(got, m)
		case <-ctx.Done():
			t.Fatal("timed out waiting for messages")
		}
	}

	require.Equal(t, []string{"hello", "world"}, got)
}

// TestSubscribeCancelStopsChannel verifies that calling the cancel function
// closes the message channel.
func TestSubscribeCancelStopsChannel(t *testing.T) {
	redisURL := redisTestURL(t)

	client, err := NewClient(redisURL)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	channel := "test:pubsub:cancel:" + t.Name()
	msgs, unsub := client.Subscribe(ctx, channel)
	unsub()

	// The message channel should be closed (or become closed shortly).
	select {
	case _, ok := <-msgs:
		require.False(t, ok, "expected channel to be closed")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel to close after cancel")
	}
}

func redisTestURL(t *testing.T) string {
	t.Helper()
	url := "redis://localhost:6379"
	// Try to connect; skip if Redis is unavailable.
	client, err := NewClient(url)
	if err != nil {
		t.Skipf("skipping: cannot parse Redis URL: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()
	if err := client.rdb.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("skipping: Redis not available at %s: %v", url, err)
	}
	_ = client.Close()
	return url
}
