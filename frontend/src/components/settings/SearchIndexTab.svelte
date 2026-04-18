<script lang="ts">
  import { rebuildSearchIndex } from "../../lib/api";
  import { Search } from "lucide-svelte";
  import Button from "../ui/Button.svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";

  let loading = $state(false);
  let error: string | null = $state(null);
  let successMessage: string | null = $state(null);

  async function handleRebuild() {
    error = null;
    successMessage = null;
    loading = true;

    try {
      const result = await rebuildSearchIndex();
      successMessage = result.message;
    } catch (err) {
      error =
        err instanceof Error ? err.message : "Failed to rebuild search index";
    } finally {
      loading = false;
    }
  }
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
>
  <div>
    <h2
      class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-1 flex items-center gap-2"
    >
      <Search class="w-5 h-5 text-accent-600" aria-hidden="true" />
      Search Index
    </h2>
    <p class="text-sm text-ink-500 dark:text-ink-300 mb-6">
      Manually trigger a full rebuild of the full-text search index. This is
      useful after bulk imports (Calibre import, library scan) that may
      temporarily leave the search index stale. On PostgreSQL this operation
      completes immediately as the search indexes are maintained automatically.
    </p>

    {#if error}
      <AlertBanner variant="error">{error}</AlertBanner>
    {/if}

    {#if successMessage}
      <AlertBanner variant="success">{successMessage}</AlertBanner>
    {/if}

    <Button
      type="button"
      onclick={handleRebuild}
      disabled={loading}
      class="mt-4"
    >
      {loading ? "Rebuilding…" : "Rebuild Search Index"}
    </Button>
  </div>
</div>
