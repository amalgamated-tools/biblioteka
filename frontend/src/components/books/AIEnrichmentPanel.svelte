<script lang="ts">
  import type { AIEnrichment } from "../../types";
  import { ApiError } from "../../lib/api/core";
  import * as api from "../../lib/api";
  import { Sparkles, Check, X } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";

  interface Props {
    bookId: string;
    onApplied?: () => void;
    onRejected?: () => void;
  }

  let { bookId, onApplied, onRejected }: Props = $props();

  let enrichment: AIEnrichment | null = $state(null);
  let loading = $state(true);
  let applying = $state(false);
  let rejecting = $state(false);
  let error: string | null = $state(null);

  $effect(() => {
    loadEnrichment(bookId);
    return () => {
      enrichment = null;
      loading = true;
      applying = false;
      rejecting = false;
      error = null;
    };
  });

  async function loadEnrichment(id: string) {
    loading = true;
    error = null;
    try {
      const result = await api.getPendingAIEnrichment(id);
      if (bookId !== id) return;
      enrichment = result;
    } catch (e) {
      if (bookId !== id) return;
      enrichment = null;
      // 404 means no pending enrichment — that's normal, render nothing
      if (!(e instanceof ApiError && e.status === 404)) {
        error = e instanceof Error ? e.message : "Failed to load AI enrichment";
      }
    } finally {
      if (bookId === id) loading = false;
    }
  }

  async function handleApply() {
    if (!enrichment) return;
    const targetBookId = bookId;
    applying = true;
    error = null;
    try {
      await api.applyAIEnrichment(targetBookId);
      if (bookId !== targetBookId) return;
      enrichment = null;
      onApplied?.();
    } catch (e) {
      if (bookId !== targetBookId) return;
      error = e instanceof Error ? e.message : "Failed to apply AI enrichment";
    } finally {
      if (bookId === targetBookId) applying = false;
    }
  }

  async function handleReject() {
    if (!enrichment) return;
    const targetBookId = bookId;
    rejecting = true;
    error = null;
    try {
      await api.rejectAIEnrichment(targetBookId);
      if (bookId !== targetBookId) return;
      enrichment = null;
      onRejected?.();
    } catch (e) {
      if (bookId !== targetBookId) return;
      error = e instanceof Error ? e.message : "Failed to reject AI enrichment";
    } finally {
      if (bookId === targetBookId) rejecting = false;
    }
  }
</script>

{#if loading}
  <div
    class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6"
  >
    <p role="status" class="text-center text-sm text-ink-500 dark:text-ink-300">
      <span
        class="inline-block w-4 h-4 rounded-full border-2 border-accent-300 dark:border-accent-600 border-t-accent-600 dark:border-t-accent-300 animate-spin mr-2 align-middle"
        aria-hidden="true"
      ></span>
      Checking for AI enrichment...
    </p>
  </div>
{:else if enrichment}
  <div
    class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-accent-200 dark:border-accent-700/40 p-6"
    data-testid="ai-enrichment-panel"
  >
    <div class="flex items-center justify-between mb-4">
      <h2
        class="text-lg font-display font-bold text-ink-900 dark:text-cream-100 flex items-center gap-2"
      >
        <Sparkles
          class="w-5 h-5 text-accent-600 dark:text-accent-400"
          aria-hidden="true"
        />
        AI Enrichment Review
      </h2>
      <div class="flex items-center gap-2">
        <Button
          onclick={handleApply}
          size="sm"
          disabled={applying || rejecting}
        >
          <Check class="w-3.5 h-3.5 mr-1" aria-hidden="true" />
          {applying ? "Applying..." : "Apply"}
        </Button>
        <Button
          variant="secondary"
          onclick={handleReject}
          size="sm"
          disabled={applying || rejecting}
        >
          <X class="w-3.5 h-3.5 mr-1" aria-hidden="true" />
          {rejecting ? "Rejecting..." : "Reject"}
        </Button>
      </div>
    </div>

    {#if error}
      <AlertBanner variant="error" class="mb-4">{error}</AlertBanner>
    {/if}

    <div class="space-y-4 text-sm">
      {#if enrichment.suggested_tags.length > 0}
        <div>
          <h3 class="text-ink-500 dark:text-ink-400 mb-2">Suggested Tags</h3>
          <div class="flex flex-wrap gap-1.5" aria-label="Suggested tags">
            {#each enrichment.suggested_tags as tag (tag)}
              <span
                class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-accent-100 text-accent-800 dark:bg-accent-800/30 dark:text-accent-300"
              >
                {tag}
              </span>
            {/each}
          </div>
        </div>
      {/if}

      {#if enrichment.reading_level}
        <div>
          <h3 class="text-ink-500 dark:text-ink-400 mb-1">Reading Level</h3>
          <p class="text-ink-700 dark:text-ink-200 font-medium">
            {enrichment.reading_level}
          </p>
        </div>
      {/if}

      {#if enrichment.generated_description}
        <div>
          <h3 class="text-ink-500 dark:text-ink-400 mb-1">
            Generated Description
          </h3>
          <p class="text-ink-700 dark:text-ink-200 whitespace-pre-line">
            {enrichment.generated_description}
          </p>
        </div>
      {/if}
    </div>

    <p class="mt-4 text-xs text-ink-400 dark:text-ink-500">
      Generated by {enrichment.provider} · {enrichment.model}
    </p>
  </div>
{:else if error}
  <AlertBanner variant="error">{error}</AlertBanner>
{/if}
