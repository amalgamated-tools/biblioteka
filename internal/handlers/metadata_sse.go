package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/pubsub"
)

// sseWriteTimeout is the per-write deadline for SSE writes; reset on every heartbeat and message.
const sseWriteTimeout = 2 * time.Minute

// sseHeartbeatInterval is the interval between SSE keepalive comments.
const sseHeartbeatInterval = 15 * time.Second

// streamEvents opens an SSE connection that streams metadata fetch progress
// events from Redis pub/sub.
//
//	@Summary		Stream metadata fetch events
//	@Description	Opens a Server-Sent Events (SSE) stream for metadata fetch progress. Events are JSON objects with an "event" field (e.g. "complete", "error", "not_found"). The connection closes automatically on a terminal event or on client disconnect. A per-write deadline of 2 minutes is enforced and reset on each heartbeat or event write.
//	@Tags			Book Metadata
//	@Produce		text/event-stream
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{string}	string	"SSE event stream"
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Failure		503	{object}	errorResponse
//	@Router			/books/{id}/metadata/events [get]
func (h *MetadataHandler) streamEvents(w http.ResponseWriter, r *http.Request, bookID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(r.Context(), w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	if h.Subscriber == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "event streaming not available")
		return
	}

	// Verify the book exists before opening a long-lived SSE connection.
	if _, err := h.DB.GetBook(r.Context(), bookID); handleDBErr(r.Context(), w, err, "book") {
		return
	}

	// Extend the write deadline for this long-lived connection.
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout)); err != nil {
		slog.WarnContext(r.Context(), "failed to set write deadline for SSE",
			slog.Any(otelkeys.Error, err),
		)
	}

	// Subscribe to Redis pub/sub BEFORE flushing headers. The client opens
	// this SSE connection and then immediately POSTs /metadata/fetch, so the
	// subscription must be confirmed before the client can trigger the job —
	// otherwise events published between Flush() and Subscribe() are lost.
	userID := auth.UserIDFromContext(r.Context())
	channel := pubsub.MetadataChannel(bookID, userID)

	msgs, cancel := h.Subscriber.Subscribe(r.Context(), channel)
	defer cancel()

	// Now that the subscription is active, send SSE headers so the client
	// knows the stream is ready and can proceed to trigger the fetch job.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
	flusher.Flush()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			// SSE comment line as keepalive.
			if err := rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout)); err != nil {
				slog.WarnContext(r.Context(), "failed to extend write deadline for SSE heartbeat",
					slog.Any(otelkeys.Error, err),
				)
			}
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case msg, ok := <-msgs:
			if !ok {
				return
			}

			// Extend deadline on each write.
			if err := rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout)); err != nil {
				slog.WarnContext(r.Context(), "failed to extend write deadline for SSE",
					slog.Any(otelkeys.Error, err),
				)
			}

			_, err := fmt.Fprintf(w, "data: %s\n\n", msg)
			if err != nil {
				return
			}
			flusher.Flush()

			// Close the connection on terminal events.
			var evt struct {
				Event string `json:"event"`
			}
			if json.Unmarshal([]byte(msg), &evt) == nil {
				if evt.Event == jobs.EventComplete || evt.Event == jobs.EventError || evt.Event == jobs.EventNotFound {
					return
				}
			}
		}
	}
}
