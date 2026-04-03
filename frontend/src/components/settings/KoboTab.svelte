<script lang="ts">
  import {
    listKoboTokens,
    createKoboToken,
    deleteKoboToken,
  } from "../../lib/api";
  import type { KoboToken } from "../../types";
  import { copyToClipboard } from "../../lib/clipboard";
  import { BookOpen, Copy, Trash2 } from "lucide-svelte";
  import { onDestroy, onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import { autofocusFirstButton } from "../../lib/actions";
  import { TokenListState } from "../../lib/tokenList.svelte";
  import { CopyTimeoutState } from "../../lib/copyTimeout.svelte";

  type KoboTokenDisplay = KoboToken & { token?: string };

  function focusTrigger(id: string) {
    document.querySelector<HTMLElement>(`[data-delete-trigger="${id}"]`)?.focus();
  }

  function cancelWithFocus() {
    const id = tokenList.pendingDelete?.id;
    void tokenList.cancelDelete(id ? () => focusTrigger(id) : undefined);
  }

  const tokenList = new TokenListState<KoboTokenDisplay>({
    load: listKoboTokens,
    delete: deleteKoboToken,
    loadError: "Failed to load Kobo tokens",
    deleteError: "Failed to delete Kobo token",
  });

  const copyState = new CopyTimeoutState();
  let newTokenName = $state("");
  let createTokenLoading = $state(false);
  let liveMessage = $state("");

  onDestroy(() => copyState.clear());

  onMount(() => {
    void tokenList.load();
  });

  async function handleCreateToken(e: SubmitEvent) {
    e.preventDefault();
    if (!newTokenName.trim()) return;

    createTokenLoading = true;
    tokenList.error = null;

    try {
      const token = await createKoboToken(newTokenName.trim());
      newTokenName = "";
      tokenList.items = [token, ...tokenList.items];
    } catch (err) {
      tokenList.error =
        err instanceof Error ? err.message : "Failed to create Kobo token";
    } finally {
      createTokenLoading = false;
    }
  }

  function syncURL(token: KoboTokenDisplay): string | null {
    if (!token.token) return null;
    return `${window.location.origin}/kobo/${token.token}/v1/initialization`;
  }

  async function handleCopyURL(
    text: string,
    tokenId: string,
    tokenName: string,
  ) {
    tokenList.error = null;

    try {
      await copyToClipboard(text);

      copyState.set(tokenId);
      liveMessage = `Copied sync URL for ${tokenName}`;
    } catch (err) {
      tokenList.error =
        err instanceof Error
          ? `Failed to copy to clipboard: ${err.message}`
          : "Failed to copy to clipboard. Your browser may not support clipboard access.";
    }
  }
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
>
  <div>
    <h2
      class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-2 flex items-center gap-2"
    >
      <BookOpen class="w-5 h-5 text-accent-600" />
      Kobo Sync
    </h2>
    <p class="text-sm text-ink-500 dark:text-ink-400 mb-6">
      Sync your library to a Kobo e-reader. Create a token below, then set the
      <strong>API endpoint</strong> in your Kobo's
      <code class="px-1 py-0.5 bg-ink-100 dark:bg-ink-800 rounded text-xs"
        >Kobo eReader.conf</code
      >
      to the sync URL shown when the token is created. Token values are only shown
      once.
    </p>

    <form onsubmit={handleCreateToken} class="flex gap-3 mb-6">
      <label for="new-kobo-token-name" class="sr-only">Token name</label>
      <TextInput
        id="new-kobo-token-name"
        bind:value={newTokenName}
        class="flex-1 py-2.5"
        placeholder="Token name (e.g., My Kobo Libra)"
        disabled={createTokenLoading}
        maxlength={100}
      />
      <Button
        type="submit"
        disabled={createTokenLoading || !newTokenName.trim()}
        class="px-5 py-2.5 whitespace-nowrap"
      >
        {createTokenLoading ? "Creating..." : "Create Token"}
      </Button>
    </form>

    {#if tokenList.error}
      <AlertBanner variant="error" class="mb-4">{tokenList.error}</AlertBanner>
    {/if}

    {#if tokenList.loading}
      <p class="text-ink-400 dark:text-ink-400">Loading Kobo tokens...</p>
    {:else if tokenList.items.length === 0}
      <p class="text-sm text-ink-400 dark:text-ink-500">
        No Kobo tokens yet. Create one above to get started.
      </p>
    {:else}
      <div class="space-y-3">
        {#each tokenList.items as token (token.id)}
          {@const url = syncURL(token)}
          <div
            class="border border-ink-100 dark:border-ink-800 rounded-xl p-4 flex flex-col gap-2 hover:bg-ink-50/50 dark:hover:bg-ink-800/30 transition-colors"
          >
            <div class="flex items-center justify-between gap-2">
              <span
                class="font-medium text-ink-900 dark:text-cream-100 truncate"
                >{token.name}</span
              >
              <div class="flex items-center gap-2 flex-shrink-0">
                <span class="text-xs text-ink-400 dark:text-ink-500"
                  >Created {new Date(
                    token.created_at,
                  ).toLocaleDateString()}</span
                >
                {#if tokenList.pendingDelete?.id === token.id}
                  <div
                    class="flex items-center gap-2 animate-scale-in"
                    role="alertdialog"
                    aria-modal="false"
                    aria-labelledby={`delete-token-confirm-label-${token.id}`}
                    tabindex="-1"
                    use:autofocusFirstButton
                    onkeydown={(e: KeyboardEvent) => {
                      if (e.key === "Escape") cancelWithFocus();
                    }}
                  >
                    <span
                      id={`delete-token-confirm-label-${token.id}`}
                      class="text-xs text-danger-600 dark:text-red-400"
                      >Delete "{token.name}"?</span
                    >
                    <Button
                      type="button"
                      variant="danger"
                      onclick={() => tokenList.confirmDelete()}
                      class="px-3 py-1 text-xs"
                    >
                      Delete
                    </Button>
                    <Button
                      type="button"
                      variant="secondary"
                      onclick={() => cancelWithFocus()}
                      class="px-3 py-1 text-xs"
                    >
                      Cancel
                    </Button>
                  </div>
                {:else}
                  <button
                    data-delete-trigger={token.id}
                    onclick={() => tokenList.handleDelete(token.id, token.name)}
                    aria-label={`Delete token ${token.name} (created ${new Date(token.created_at).toLocaleDateString()})`}
                    class="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium text-danger-600 hover:bg-danger-50 dark:text-red-400 dark:hover:bg-danger-700/10 transition-colors"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                    Delete
                  </button>
                {/if}
              </div>
            </div>

            <div class="flex items-center gap-2">
              {#if url}
                <code
                  class="flex-1 px-3 py-2 bg-ink-50 dark:bg-ink-800 border border-ink-100 dark:border-ink-700 rounded-lg text-xs font-mono text-ink-600 dark:text-ink-300 break-all"
                >
                  {url}
                </code>
                <button
                  onclick={() => handleCopyURL(url, token.id, token.name)}
                  aria-label={copyState.copiedId === token.id
                    ? `Copied sync URL for ${token.name}`
                    : `Copy sync URL for ${token.name}`}
                  class="flex-shrink-0 flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {copyState.copiedId ===
                  token.id
                    ? 'bg-success-100 text-success-700 dark:bg-green-900/40 dark:text-green-400'
                    : 'bg-ink-100 text-ink-600 hover:bg-ink-200 dark:bg-ink-800 dark:text-ink-300 dark:hover:bg-ink-700'}"
                >
                  <Copy class="w-4 h-4" />
                  {copyState.copiedId === token.id ? "Copied" : "Copy"}
                </button>
              {:else}
                <div
                  class="flex-1 px-3 py-2 bg-ink-50 dark:bg-ink-800 border border-ink-100 dark:border-ink-700 rounded-lg text-xs text-ink-500 dark:text-ink-400"
                >
                  Token hidden. Create a new token to get a fresh sync URL.
                </div>
                <button
                  type="button"
                  aria-disabled="true"
                  aria-label={`Copy unavailable for ${token.name} — token value is only shown once`}
                  onclick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                  }}
                  class="flex-shrink-0 flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium bg-ink-100 text-ink-400 dark:bg-ink-800 dark:text-ink-500 cursor-not-allowed"
                >
                  <Copy class="w-4 h-4" aria-hidden="true" />
                  Copy
                </button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}

    <span role="status" class="sr-only">{liveMessage}</span>
  </div>

  <div
    class="border-t border-ink-100 dark:border-ink-800 pt-4 text-sm text-ink-500 dark:text-ink-400 space-y-2"
  >
    <p class="font-medium text-ink-700 dark:text-ink-300">Setup instructions</p>
    <ol class="list-decimal list-inside space-y-1">
      <li>Create a Kobo sync token above.</li>
      <li>
        Connect your Kobo to a computer and open
        <code class="px-1 py-0.5 bg-ink-100 dark:bg-ink-800 rounded"
          >.kobo/Kobo/Kobo eReader.conf</code
        >.
      </li>
      <li>
        Under <code class="px-1 py-0.5 bg-ink-100 dark:bg-ink-800 rounded"
          >[OneStoreServices]</code
        >, add or update:<br />
        <code class="px-1 py-0.5 bg-ink-100 dark:bg-ink-800 rounded"
          >api_endpoint=&lt;your sync URL&gt;</code
        >
      </li>
      <li>Safely eject the Kobo and trigger a sync on the device.</li>
    </ol>
    <p class="text-xs mt-2">
      Only EPUB, KEPUB, MOBI, PDF, and AZW3 files are synced. HTTPS is
      recommended for external access.
    </p>
  </div>
</div>
