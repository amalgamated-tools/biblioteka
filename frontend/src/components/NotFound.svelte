<script lang="ts">
  import { routerStore } from "../stores/router.svelte";
</script>

<div class="text-center py-16 animate-fade-in">
  <p
    class="text-8xl font-display font-bold text-ink-200 dark:text-ink-800 mb-4 select-none"
    aria-hidden="true"
  >
    404
  </p>
  <h1
    class="text-2xl font-display font-bold text-ink-900 dark:text-cream-100 mb-2"
  >
    Page Not Found
  </h1>
  <p class="text-ink-400 dark:text-ink-500 mb-8">
    The page you are looking for does not exist.
  </p>
  <button
    onclick={() => routerStore.navigate("dashboard")}
    class="inline-flex items-center gap-2 px-5 py-2.5 bg-accent-600 hover:bg-accent-700 text-white rounded-xl font-medium transition-colors"
  >
    Go to Dashboard
  </button>
</div>
