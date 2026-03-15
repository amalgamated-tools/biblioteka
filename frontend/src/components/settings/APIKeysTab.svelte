<script lang="ts">
  import {
    listAPIKeys,
    createAPIKey,
    deleteAPIKey,
    type APIKey,
  } from "../../lib/api";
  import { KeyRound, Copy, Trash2 } from "lucide-svelte";

  let apiKeyList: APIKey[] = $state.raw([]);
  let apiKeysLoading = $state(false);
  let apiKeysError: string | null = $state(null);
  let newKeyName = $state("");
  let newlyCreatedKey: string | null = $state(null);
  let createKeyLoading = $state(false);
  let apiKeysLoaded = $state(false);
  let keyCopied = $state(false);
  let apiKeysTried = $state(false);

  $effect(() => {
    if (!apiKeysLoaded && !apiKeysLoading && !apiKeysTried) {
      apiKeysTried = true;
      loadAPIKeys().then(() => {
        if (!apiKeysLoaded && apiKeysError) {
          apiKeysTried = false;
        }
      });
    }
  });

  async function loadAPIKeys() {
    apiKeysLoading = true;
    apiKeysError = null;
    try {
      apiKeyList = await listAPIKeys();
      apiKeysLoaded = true;
    } catch (err) {
      apiKeysError =
        err instanceof Error ? err.message : "Failed to load API keys";
    } finally {
      apiKeysLoading = false;
    }
  }

  async function handleCreateAPIKey(e: SubmitEvent) {
    e.preventDefault();
    if (!newKeyName.trim()) return;

    createKeyLoading = true;
    apiKeysError = null;
    newlyCreatedKey = null;

    try {
      const result = await createAPIKey(newKeyName.trim());
      newlyCreatedKey = result.key;
      newKeyName = "";
      apiKeyList = [
        {
          id: result.id,
          name: result.name,
          key_prefix: result.key_prefix,
          last_used_at: result.last_used_at,
          created_at: result.created_at,
        },
        ...apiKeyList,
      ];
    } catch (err) {
      apiKeysError =
        err instanceof Error ? err.message : "Failed to create API key";
    } finally {
      createKeyLoading = false;
    }
  }

  async function handleDeleteAPIKey(id: string) {
    apiKeysError = null;
    try {
      await deleteAPIKey(id);
      apiKeyList = apiKeyList.filter((k) => k.id !== id);
    } catch (err) {
      apiKeysError =
        err instanceof Error ? err.message : "Failed to delete API key";
    }
  }

  async function copyToClipboard(text: string) {
    try {
      await navigator.clipboard.writeText(text);
      keyCopied = true;
      setTimeout(() => (keyCopied = false), 2000);
    } catch {
      // Clipboard API unavailable (insecure context or denied permission).
      // Fallback: use a temporary textarea to select+copy the text.
      const textarea = document.createElement("textarea");
      textarea.value = text;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.appendChild(textarea);
      textarea.select();
      try {
        document.execCommand("copy");
        keyCopied = true;
        setTimeout(() => (keyCopied = false), 2000);
      } catch {
        apiKeysError =
          "Failed to copy to clipboard. Please select and copy the key manually.";
      } finally {
        document.body.removeChild(textarea);
      }
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
      <input
        type="text"
        bind:value={newKeyName}
        class="flex-1 px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
        placeholder="Key name (e.g., CI Pipeline)"
        disabled={createKeyLoading}
        maxlength={100}
      />
      <button
        type="submit"
        disabled={createKeyLoading || !newKeyName.trim()}
        class="px-5 py-2.5 bg-gradient-to-r from-accent-600 to-accent-700 text-white font-semibold rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all disabled:opacity-50 shadow-md shadow-accent-600/20 whitespace-nowrap"
      >
        {createKeyLoading ? "Creating..." : "Create Key"}
      </button>
    </form>

    {#if newlyCreatedKey}
      <div
        class="bg-success-50 dark:bg-green-900/20 border border-success-600/20 dark:border-green-700/30 px-4 py-4 rounded-xl mb-6 animate-scale-in"
      >
        <p
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
              try {
                await copyToClipboard(newlyCreatedKey!);
                keyCopied = true;
                newlyCreatedKey = null;
              } catch (error) {
                apiKeysError =
                  "Failed to copy API key. Please copy it manually or try again.";
              }
            }}
            class="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {keyCopied
              ? 'bg-success-100 text-success-700 dark:bg-green-900/40 dark:text-green-400'
              : 'bg-ink-100 text-ink-600 hover:bg-ink-200 dark:bg-ink-800 dark:text-ink-300 dark:hover:bg-ink-700'}"
          >
            <Copy class="w-4 h-4" />
            {keyCopied ? "Copied" : "Copy"}
          </button>
        </div>
      </div>
    {/if}

    {#if apiKeysError}
      <div
        class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm mb-4"
      >
        {apiKeysError}
      </div>
    {/if}

    {#if apiKeysLoading}
      <p class="text-ink-400 dark:text-ink-400">Loading API keys...</p>
    {:else if apiKeyList.length === 0}
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
              <th class="pb-3 font-medium">Name</th>
              <th class="pb-3 font-medium">Key</th>
              <th class="pb-3 font-medium">Created</th>
              <th class="pb-3 font-medium">Last Used</th>
              <th class="pb-3 font-medium"></th>
            </tr>
          </thead>
          <tbody>
            {#each apiKeyList as key (key.id)}
              <tr
                class="border-b border-ink-50 dark:border-ink-800 hover:bg-ink-50/50 dark:hover:bg-ink-800/50 transition-colors"
              >
                <td
                  class="py-3 text-ink-900 dark:text-cream-100 font-medium"
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
                  <button
                    onclick={() => handleDeleteAPIKey(key.id)}
                    class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-danger-600 hover:bg-danger-50 dark:text-red-400 dark:hover:bg-danger-700/10 transition-colors"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                    Delete
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
