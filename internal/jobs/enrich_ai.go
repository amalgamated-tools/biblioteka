package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/llm"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/ptrutil"
	"github.com/amalgamated-tools/biblioteka/internal/pubsub"
)

// JobEnrichAI is the registered name for the AI enrichment job.
const JobEnrichAI = "enrich:ai"

// EnrichAIPayload is the JSON payload for the enrich:ai job.
type EnrichAIPayload struct {
	BookID string `json:"book_id"`
	UserID string `json:"user_id"`
}

// NewEnrichAIHandler returns a job handler that fetches book metadata using
// an LLM provider and stores the result as a pending AIEnrichment.
func NewEnrichAIHandler(database *db.DB, provider llm.Provider, providerName string, publisher pubsub.Publisher) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, payload []byte) error {
		var p EnrichAIPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			slog.ErrorContext(ctx, "failed to unmarshal enrich:ai payload",
				slog.Any(otelkeys.Error, err),
			)
			return fmt.Errorf("unmarshal enrich ai payload: %w", err)
		}

		slog.DebugContext(ctx, "enrich:ai job received",
			slog.String(otelkeys.BookID, p.BookID),
			slog.String(otelkeys.UserID, p.UserID),
		)

		if provider == nil {
			slog.WarnContext(ctx, "enrich:ai job skipped: no LLM provider configured",
				slog.String(otelkeys.BookID, p.BookID),
			)
			channel := pubsub.MetadataChannel(p.BookID, p.UserID)
			publishEvent(ctx, publisher, channel, progressEvent{Event: EventError, Source: providerName, Message: "LLM provider not configured"})
			return nil
		}

		return enrichAI(ctx, database, provider, providerName, publisher, p)
	}
}

func enrichAI(ctx context.Context, database *db.DB, provider llm.Provider, providerName string, publisher pubsub.Publisher, p EnrichAIPayload) error {
	channel := pubsub.MetadataChannel(p.BookID, p.UserID)

	publishAIProgress(ctx, publisher, channel, providerName, "fetching_book", "Fetching book...")

	book, err := database.GetBook(ctx, p.BookID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch book for AI enrichment",
			slog.String(otelkeys.BookID, p.BookID),
			slog.Any(otelkeys.Error, err),
		)
		publishEvent(ctx, publisher, channel, progressEvent{Event: EventError, Source: providerName, Message: "failed to fetch book"})
		return fmt.Errorf("fetch book %s: %w", p.BookID, err)
	}

	authors, err := database.GetBookAuthors(ctx, p.BookID)
	if err != nil {
		slog.WarnContext(ctx, "failed to fetch book authors for AI enrichment",
			slog.String(otelkeys.BookID, p.BookID),
			slog.Any(otelkeys.Error, err),
		)
	}

	authorNames := make([]string, 0, len(authors))
	for _, a := range authors {
		authorNames = append(authorNames, a.Name)
	}

	publishAIProgress(ctx, publisher, channel, providerName, "building_prompt", "Building prompt...")

	prompt := llm.BuildEnrichPrompt(book.Title, authorNames, ptrutil.Deref(book.Description))

	publishAIProgress(ctx, publisher, channel, providerName, "generating", "Generating metadata...")

	raw, err := provider.Generate(ctx, prompt)
	if err != nil {
		slog.ErrorContext(ctx, "LLM generation failed",
			slog.String(otelkeys.BookID, p.BookID),
			slog.Any(otelkeys.Error, err),
		)
		publishEvent(ctx, publisher, channel, progressEvent{Event: EventError, Source: providerName, Message: "LLM generation failed"})
		return fmt.Errorf("llm generate for book %s: %w", p.BookID, err)
	}

	result, err := llm.ParseEnrichmentResult(raw)
	if err != nil {
		slog.ErrorContext(ctx, "failed to parse LLM enrichment result",
			slog.String(otelkeys.BookID, p.BookID),
			slog.Any(otelkeys.Error, err),
		)
		publishEvent(ctx, publisher, channel, progressEvent{Event: EventError, Source: providerName, Message: "failed to parse LLM response"})
		return fmt.Errorf("parse enrichment result for book %s: %w", p.BookID, err)
	}

	bookIDPtr := ptrutil.NilIfZero(p.BookID)
	var readingLevel *string
	if result.ReadingLevel != "" {
		readingLevel = &result.ReadingLevel
	}
	var generatedDesc *string
	if result.GeneratedDescription != "" {
		generatedDesc = &result.GeneratedDescription
	}

	enrichment, err := database.CreateAIEnrichment(ctx, p.UserID, bookIDPtr, providerName, "", result.SuggestedTags, readingLevel, generatedDesc, raw)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create AI enrichment",
			slog.String(otelkeys.BookID, p.BookID),
			slog.Any(otelkeys.Error, err),
		)
		publishEvent(ctx, publisher, channel, progressEvent{Event: EventError, Source: providerName, Message: "failed to save enrichment"})
		return fmt.Errorf("create AI enrichment for book %s: %w", p.BookID, err)
	}

	publishEvent(ctx, publisher, channel, progressEvent{Event: EventComplete, Source: providerName, MetadataID: enrichment.ID})

	return nil
}

// publishAIProgress is a convenience wrapper that publishes a progress event
// with the given source (provider name).
func publishAIProgress(ctx context.Context, publisher pubsub.Publisher, channel, source, step, message string) {
	publishEvent(ctx, publisher, channel, progressEvent{
		Event:   EventProgress,
		Source:  source,
		Step:    step,
		Message: message,
	})
}
