// Package pubsub provides Redis-backed publish/subscribe messaging for
// broadcasting real-time job progress events from background workers to
// SSE-connected HTTP clients.
package pubsub

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/redis/go-redis/v9"
)

// metadataChannelPrefix is the Redis pub/sub channel prefix for metadata fetch
// progress events.
const metadataChannelPrefix = "biblioteka:metadata:"

// Publisher sends messages to a named channel.
type Publisher interface {
	Publish(ctx context.Context, channel, message string) error
}

// Subscriber listens for messages on a named channel.
type Subscriber interface {
	// Subscribe returns a channel that receives messages published to the given
	// Redis channel, and a cancel function that unsubscribes and closes the
	// message channel. The caller must call cancel when done.
	Subscribe(ctx context.Context, channel string) (<-chan string, func())
}

// Client implements both Publisher and Subscriber using a Redis connection.
type Client struct {
	rdb *redis.Client
}

// NewClient creates a pub/sub client connected to the Redis instance at the
// given URL (e.g. "redis://localhost:6379").
func NewClient(redisURL string) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}
	return &Client{rdb: redis.NewClient(opts)}, nil
}

// Publish sends a message to the given channel.
func (c *Client) Publish(ctx context.Context, channel, message string) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

// Subscribe listens for messages on the given channel. It returns a string
// channel and a cancel function. The caller must call cancel to release
// resources. The returned channel is closed when cancel is called or the
// context is canceled.
func (c *Client) Subscribe(ctx context.Context, channel string) (<-chan string, func()) {
	sub := c.rdb.Subscribe(ctx, channel)

	// Wait for the server to acknowledge the subscription so that callers
	// don't miss messages published immediately after Subscribe returns.
	if _, err := sub.Receive(ctx); err != nil {
		slog.WarnContext(ctx, "failed to confirm Redis subscription",
			slog.String(otelkeys.PubSubChannel, channel),
			slog.Any(otelkeys.Error, err),
		)
	}

	ch := make(chan string, 1)
	done := make(chan struct{})
	var once sync.Once

	go func() {
		defer close(ch)
		redisCh := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case msg, ok := <-redisCh:
				if !ok {
					return
				}
				select {
				case ch <- msg.Payload:
				case <-ctx.Done():
					return
				case <-done:
					return
				}
			}
		}
	}()

	cancel := func() {
		once.Do(func() {
			close(done)
			if err := sub.Close(); err != nil {
				slog.WarnContext(ctx, "failed to close Redis subscription",
					slog.String(otelkeys.PubSubChannel, channel),
					slog.Any(otelkeys.Error, err),
				)
			}
		})
	}

	return ch, cancel
}

// Close closes the underlying Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// MetadataChannel returns the pub/sub channel name for metadata fetch progress
// events for a given book and user.
func MetadataChannel(bookID, userID string) string {
	return metadataChannelPrefix + bookID + ":" + userID
}
