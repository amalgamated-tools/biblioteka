package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

// mockLLMProvider implements llm.Provider for testing.
type mockLLMProvider struct {
	response string
	err      error
}

func (m *mockLLMProvider) Generate(_ context.Context, _ string) (string, error) {
	return m.response, m.err
}

// mockPublisher captures published events for assertions.
type mockPublisher struct {
	messages []string
}

func (m *mockPublisher) Publish(_ context.Context, _, message string) error {
	m.messages = append(m.messages, message)
	return nil
}

func (m *mockPublisher) hasEvent(eventType string) bool {
	for _, msg := range m.messages {
		var evt struct {
			Event string `json:"event"`
		}
		if json.Unmarshal([]byte(msg), &evt) == nil && evt.Event == eventType {
			return true
		}
	}
	return false
}

func TestEnrichAI_Success(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err)

	book, err := d.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err)

	validJSON := `{"genres":["Science Fiction"],"themes":["Power","Religion"],"mood":"Epic","reading_level":"adult","suggested_tags":["sci-fi","classic","dune"],"generated_description":"A sweeping epic set in a desert world."}`
	provider := &mockLLMProvider{response: validJSON}
	publisher := &mockPublisher{}

	payload, err := json.Marshal(EnrichAIPayload{BookID: book.ID, UserID: user.ID})
	require.NoError(t, err)

	handler := NewEnrichAIHandler(d, provider, "ollama", publisher)
	err = handler(t.Context(), payload)
	require.NoError(t, err)

	// Verify enrichment was created
	enrichment, err := d.GetPendingAIEnrichmentByBook(t.Context(), user.ID, book.ID)
	require.NoError(t, err)
	require.Equal(t, db.AIEnrichmentStatusPending, enrichment.Status)
	require.Equal(t, "ollama", enrichment.Provider)
	require.Contains(t, enrichment.SuggestedTags, "sci-fi")

	// Verify complete event was published
	require.True(t, publisher.hasEvent(EventComplete))
}

func TestEnrichAI_NilProvider(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err)

	book, err := d.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err)

	publisher := &mockPublisher{}

	payload, err := json.Marshal(EnrichAIPayload{BookID: book.ID, UserID: user.ID})
	require.NoError(t, err)

	handler := NewEnrichAIHandler(d, nil, "ollama", publisher)
	err = handler(t.Context(), payload)
	require.NoError(t, err) // nil provider is not a retriable error

	require.True(t, publisher.hasEvent(EventError))
}

func TestEnrichAI_ProviderError(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err)

	book, err := d.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err)

	provider := &mockLLMProvider{err: errors.New("connection refused")}
	publisher := &mockPublisher{}

	payload, err := json.Marshal(EnrichAIPayload{BookID: book.ID, UserID: user.ID})
	require.NoError(t, err)

	handler := NewEnrichAIHandler(d, provider, "ollama", publisher)
	err = handler(t.Context(), payload)
	require.Error(t, err) // provider error should be returned for retry

	require.True(t, publisher.hasEvent(EventError))
}
