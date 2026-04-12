<script lang="ts">
  import { onMount } from "svelte";
  import { Mail, X } from "lucide-svelte";
  import * as api from "../../lib/api";
  import type { Book, BookFile } from "../../types";
  import AlertBanner from "./AlertBanner.svelte";
  import Button from "./Button.svelte";

  interface Props {
    bookId: string;
    onClose: () => void;
  }

  let { bookId, onClose }: Props = $props();

  let book: Book | null = $state(null);
  let loadError: string | null = $state(null);
  let loading = $state(true);
  let dialogEl: HTMLDivElement | null = $state(null);

  onMount(() => {
    dialogEl?.focus();
  });

  let selectedFileId: string = $state("");
  let toAddress: string = $state("");
  let sending = $state(false);
  let sendError: string | null = $state(null);
  let successMessage: string | null = $state(null);

  $effect(() => {
    let active = true;

    loading = true;
    loadError = null;

    api
      .getBook(bookId)
      .then((b) => {
        if (!active) return;
        book = b;
        if (b.files.length > 0) {
          selectedFileId = b.files[0].id;
        }
      })
      .catch((e) => {
        if (!active) return;
        loadError =
          e instanceof Error ? e.message : "Failed to load book details";
      })
      .finally(() => {
        if (!active) return;
        loading = false;
      });

    return () => {
      active = false;
    };
  });

  function fileLabel(f: BookFile): string {
    return `${f.file_name} (${f.file_type.toUpperCase()})`;
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!selectedFileId || !toAddress.trim()) return;

    sending = true;
    sendError = null;
    successMessage = null;

    try {
      const result = await api.emailBookFile(selectedFileId, toAddress.trim());
      successMessage = result.message;
    } catch (err) {
      sendError = err instanceof Error ? err.message : "Failed to send email";
    } finally {
      sending = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") {
      onClose();
    }
  }
</script>

<!-- Modal wrapper -->
<svelte:window onkeydown={handleKeydown} />
<div
  bind:this={dialogEl}
  role="dialog"
  aria-modal="true"
  aria-labelledby="email-modal-title"
  tabindex="-1"
  class="fixed inset-0 z-50 flex items-center justify-center p-4"
>
  <!-- Backdrop -->
  <div
    class="absolute inset-0 bg-black/50"
    onclick={onClose}
    aria-hidden="true"
  ></div>

  <!-- Modal panel -->
  <div
    class="relative bg-white dark:bg-ink-900 rounded-2xl shadow-xl border border-ink-100 dark:border-ink-800 w-full max-w-md animate-fade-in"
  >
    <!-- Header -->
    <div
      class="flex items-center justify-between px-6 py-4 border-b border-ink-100 dark:border-ink-800"
    >
      <div class="flex items-center gap-2">
        <Mail
          class="w-5 h-5 text-accent-600 dark:text-accent-400"
          aria-hidden="true"
        />
        <h2
          id="email-modal-title"
          class="font-semibold text-ink-900 dark:text-cream-100"
        >
          Email Book
        </h2>
      </div>
      <button
        onclick={onClose}
        aria-label="Close"
        class="p-1 rounded-lg text-ink-400 hover:text-ink-600 dark:hover:text-ink-200 transition-colors"
      >
        <X class="w-5 h-5" aria-hidden="true" />
      </button>
    </div>

    <!-- Body -->
    <div class="px-6 py-5">
      {#if loading}
        <p role="status" class="text-ink-500 dark:text-ink-300 text-sm">
          Loading book details…
        </p>
      {:else if loadError}
        <AlertBanner variant="error">{loadError}</AlertBanner>
      {:else if !book || book.files.length === 0}
        <p class="text-ink-500 dark:text-ink-300 text-sm">
          This book has no files available to email.
        </p>
      {:else if successMessage}
        <AlertBanner variant="success">{successMessage}</AlertBanner>
        <div class="mt-4 flex justify-end">
          <Button variant="secondary" onclick={onClose}>Close</Button>
        </div>
      {:else}
        {#if sendError}
          <AlertBanner variant="error" class="mb-4">{sendError}</AlertBanner>
        {/if}

        <form onsubmit={handleSubmit} class="space-y-4">
          {#if book.files.length > 1}
            <div>
              <label
                for="email-file-select"
                class="block text-sm font-medium text-ink-700 dark:text-ink-300 mb-1"
              >
                File
              </label>
              <select
                id="email-file-select"
                bind:value={selectedFileId}
                class="w-full px-4 py-2 border border-ink-400 dark:border-ink-400 rounded-xl bg-white dark:bg-ink-800 text-ink-900 dark:text-cream-100 focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none"
              >
                {#each book.files as f (f.id)}
                  <option value={f.id}>{fileLabel(f)}</option>
                {/each}
              </select>
            </div>
          {:else}
            <p class="text-sm text-ink-600 dark:text-ink-300">
              <span class="font-medium">File:</span>
              {fileLabel(book.files[0])}
            </p>
          {/if}

          <div>
            <label
              for="email-to-input"
              class="block text-sm font-medium text-ink-700 dark:text-ink-300 mb-1"
            >
              To
            </label>
            <input
              id="email-to-input"
              type="email"
              bind:value={toAddress}
              placeholder="reader@example.com"
              required
              autocomplete="email"
              class="w-full px-4 py-2 border border-ink-400 dark:border-ink-400 rounded-xl bg-white dark:bg-ink-800 text-ink-900 dark:text-cream-100 focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none placeholder:text-ink-500 dark:placeholder:text-ink-300"
            />
          </div>

          <div class="flex justify-end gap-3 pt-1">
            <Button variant="secondary" type="button" onclick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={sending || !toAddress.trim()}>
              {sending ? "Sending…" : "Send"}
            </Button>
          </div>
        </form>
      {/if}
    </div>
  </div>
</div>
