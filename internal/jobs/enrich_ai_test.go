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

	handler := NewEnrichAIHandler(d, provider, "ollama", "llama3", publisher)
	err = handler(t.Context(), payload)
	require.NoError(t, err)

	// Verify enrichment was created
	enrichment, err := d.GetPendingAIEnrichmentByBook(t.Context(), user.ID, book.ID)
	require.NoError(t, err)
	require.Equal(t, db.AIEnrichmentStatusPending, enrichment.Status)
	require.Equal(t, "ollama", enrichment.Provider)
	require.Equal(t, "llama3", enrichment.Model)
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

	handler := NewEnrichAIHandler(d, nil, "ollama", "llama3", publisher)
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

	handler := NewEnrichAIHandler(d, provider, "ollama", "llama3", publisher)
	err = handler(t.Context(), payload)
	require.Error(t, err) // provider error should be returned for retry

	require.True(t, publisher.hasEvent(EventError))
}

func TestEnrichAI_InvalidPayload(t *testing.T) {
	d := newTestDB(t)
	publisher := &mockPublisher{}

	handler := NewEnrichAIHandler(d, &mockLLMProvider{}, "ollama", "llama3", publisher)
	err := handler(t.Context(), []byte("not valid json"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal")
	// Invalid payload is a permanent failure — no event is published.
	require.False(t, publisher.hasEvent(EventError))
}

func TestEnrichAI_BookNotFound(t *testing.T) {
	d := newTestDB(t)
	publisher := &mockPublisher{}

	payload, err := json.Marshal(EnrichAIPayload{BookID: "nonexistent-book-id", UserID: "any-user"})
	require.NoError(t, err)

	handler := NewEnrichAIHandler(d, &mockLLMProvider{}, "ollama", "llama3", publisher)
	err = handler(t.Context(), payload)
	require.Error(t, err)
	require.Contains(t, err.Error(), "fetch book")

	require.True(t, publisher.hasEvent(EventError))
}

func TestEnrichAI_ParseError(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err)

	book, err := d.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err)

	// LLM returns a response that cannot be parsed as an EnrichmentResult.
	provider := &mockLLMProvider{response: "Sorry, I cannot help with that."}
	publisher := &mockPublisher{}

	payload, err := json.Marshal(EnrichAIPayload{BookID: book.ID, UserID: user.ID})
	require.NoError(t, err)

	handler := NewEnrichAIHandler(d, provider, "ollama", "llama3", publisher)
	err = handler(t.Context(), payload)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse enrichment result")

	require.True(t, publisher.hasEvent(EventError))
}

func TestEnrichAI_EmptyReadingLevelAndDescription(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err)

	book, err := d.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err)

	// LLM returns a response with empty reading_level and generated_description —
	// both should be stored as nil rather than empty strings.
	sparseJSON := `{"genres":["Fiction"],"themes":[],"mood":"","reading_level":"","suggested_tags":["fiction"],"generated_description":""}`
	provider := &mockLLMProvider{response: sparseJSON}
	publisher := &mockPublisher{}

	payload, err := json.Marshal(EnrichAIPayload{BookID: book.ID, UserID: user.ID})
	require.NoError(t, err)

	handler := NewEnrichAIHandler(d, provider, "ollama", "llama3", publisher)
	err = handler(t.Context(), payload)
	require.NoError(t, err)

	enrichment, err := d.GetPendingAIEnrichmentByBook(t.Context(), user.ID, book.ID)
	require.NoError(t, err)
	require.Nil(t, enrichment.ReadingLevel, "empty reading_level should be stored as nil")
	require.Nil(t, enrichment.GeneratedDescription, "empty generated_description should be stored as nil")
	require.True(t, publisher.hasEvent(EventComplete))
}
