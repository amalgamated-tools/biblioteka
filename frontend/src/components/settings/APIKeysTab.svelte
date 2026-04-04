<script lang="ts">
  import { listAPIKeys, createAPIKey, deleteAPIKey } from "../../lib/api";
  import type { APIKey } from "../../types";
  import { copyToClipboard } from "../../lib/clipboard";
  import { KeyRound, Copy, Trash2 } from "lucide-svelte";
  import { onDestroy, onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import DeleteConfirmation from "../ui/DeleteConfirmation.svelte";
  import { createTokenManager } from "../../lib/tokenManager.svelte";
  import { CopyTimeoutState } from "../../lib/copyTimeout.svelte";

  const mgr = createTokenManager<APIKey>({
    loadFn: listAPIKeys,
    deleteFn: deleteAPIKey,
    loadErrorMessage: "Failed to load API keys",
    deleteErrorMessage: "Failed to delete API key",
  });

  let newKeyName = $state("");
  let newlyCreatedKey: string | null = $state(null);
  let createKeyLoading = $state(false);
  const newKeyCopyState = new CopyTimeoutState();

  onDestroy(() => newKeyCopyState.clear());
  onDestroy(mgr.clearCopyTimeout);

  onMount(() => {
    void mgr.load();
  });

  async function handleCreateAPIKey(e: SubmitEvent) {
    e.preventDefault();
    if (!newKeyName.trim()) return;

    createKeyLoading = true;
    mgr.error = null;
    newlyCreatedKey = null;
    // Reset any previous "copied" state when starting to create a new key
    newKeyCopyState.clear();

    try {
      const result = await createAPIKey(newKeyName.trim());
      newlyCreatedKey = result.key;
      newKeyName = "";
      mgr.items = [
        {
          id: result.id,
          name: result.name,
          key_prefix: result.key_prefix,
          last_used_at: result.last_used_at,
          created_at: result.created_at,
        },
        ...mgr.items,
      ];
    } catch (err) {
      mgr.error =
        err instanceof Error ? err.message : "Failed to create API key";
    } finally {
      createKeyLoading = false;
    }
  }

  async function handleCopyKey(text: string): Promise<boolean> {
    try {
      await copyToClipboard(text);
      newKeyCopyState.set("new-key");
      return true;
    } catch {
      mgr.error =
        "Failed to copy to clipboard. Please select and copy the key manually.";
      return false;
    }
  }
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
>
  <div>
    <h2
      class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
    >
      <KeyRound class="w-5 h-5 text-accent-600" />
      API Keys
    </h2>
    <p class="text-sm text-ink-500 dark:text-ink-400 mb-6">
      Create API keys to authenticate programmatic requests. Keys are shown only
      once at creation.
    </p>

    <form onsubmit={handleCreateAPIKey} class="flex gap-3 mb-6">
      <label for="new-api-key-name" class="sr-only">API key name</label>
      <TextInput
        id="new-api-key-name"
        bind:value={newKeyName}
        class="flex-1 py-2.5"
        placeholder="Key name (e.g., CI Pipeline)"
        disabled={createKeyLoading}
        maxlength={100}
      />
      <Button
        type="submit"
        disabled={createKeyLoading || !newKeyName.trim()}
        class="px-5 py-2.5 whitespace-nowrap"
      >
        {createKeyLoading ? "Creating..." : "Create Key"}
      </Button>
    </form>

    {#if newlyCreatedKey}
      <div
        class="bg-success-50 dark:bg-green-900/20 border border-success-600/20 dark:border-green-700/30 px-4 py-4 rounded-xl mb-6 animate-scale-in"
      >
        <p
          role="alert"
          class="text-sm font-medium text-success-700 dark:text-green-400 mb-2"
        >
          API key created successfully. Copy it now — it will not be shown
          again.
        </p>
        <div class="flex items-center gap-2">
          <code
            class="flex-1 px-3 py-2 bg-white dark:bg-ink-900 border border-success-200 dark:border-green-800 rounded-lg text-sm font-mono text-ink-900 dark:text-cream-100 break-all"
          >
            {newlyCreatedKey}
          </code>
          <button
            onclick={async () => {
              const ok = await handleCopyKey(newlyCreatedKey!);
              if (ok) {
                newlyCreatedKey = null;
              }
            }}
            class="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {newKeyCopyState.copiedId !== null
              ? 'bg-success-100 text-success-700 dark:bg-green-900/40 dark:text-green-400'
              : 'bg-ink-100 text-ink-600 hover:bg-ink-200 dark:bg-ink-800 dark:text-ink-300 dark:hover:bg-ink-700'}"
          >
            <Copy class="w-4 h-4" />
            {newKeyCopyState.copiedId !== null ? "Copied" : "Copy"}
          </button>
        </div>
      </div>
    {/if}

    {#if mgr.error}
      <AlertBanner variant="error" class="mb-4">{mgr.error}</AlertBanner>
    {/if}

    {#if mgr.loading}
      <p class="text-ink-400 dark:text-ink-400">Loading API keys...</p>
    {:else if mgr.items.length === 0}
      <p class="text-sm text-ink-400 dark:text-ink-500">
        No API keys yet. Create one above to get started.
      </p>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr
              class="text-left text-ink-400 dark:text-ink-400 border-b border-ink-100 dark:border-ink-800"
            >
              <th scope="col" class="pb-3 font-medium">Name</th>
              <th scope="col" class="pb-3 font-medium">Key</th>
              <th scope="col" class="pb-3 font-medium">Created</th>
              <th scope="col" class="pb-3 font-medium">Last Used</th>
              <th scope="col" class="pb-3 font-medium">
                <span class="sr-only">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {#each mgr.items as key (key.id)}
              <tr
                class="border-b border-ink-50 dark:border-ink-800 hover:bg-ink-50/50 dark:hover:bg-ink-800/50 transition-colors"
              >
                <td class="py-3 text-ink-900 dark:text-cream-100 font-medium"
                  >{key.name}</td
                >
                <td class="py-3">
                  <code
                    class="px-2 py-1 bg-ink-50 dark:bg-ink-800 rounded text-xs font-mono text-ink-500 dark:text-ink-400"
                  >
                    {key.key_prefix}...
                  </code>
                </td>
                <td class="py-3 text-ink-400 dark:text-ink-500"
                  >{new Date(key.created_at).toLocaleDateString()}</td
                >
                <td class="py-3 text-ink-400 dark:text-ink-500">
                  {key.last_used_at
                    ? new Date(key.last_used_at).toLocaleDateString()
                    : "Never"}
                </td>
                <td class="py-3 text-right">
                  {#if mgr.pendingDelete?.id === key.id}
                    <DeleteConfirmation
                      itemId={key.id}
                      itemName={key.name}
                      onConfirm={mgr.confirmDelete}
                      onCancel={mgr.cancelDelete}
                    />
                  {:else}
                    <button
                      data-delete-trigger={key.id}
                      onclick={() => mgr.handleDelete(key.id, key.name)}
                      aria-label={`Delete API key ${key.name} (${key.key_prefix}...)`}
                      class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-danger-600 hover:bg-danger-50 dark:text-red-400 dark:hover:bg-danger-700/10 transition-colors"
                    >
                      <Trash2 class="w-3.5 h-3.5" />
                      Delete
                    </button>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
