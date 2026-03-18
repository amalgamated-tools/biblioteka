<script lang="ts">
  import { Library } from "lucide-svelte";
</script>

<div>
  <div class="flex items-center gap-3 mb-8">
    <div
      class="w-10 h-10 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center"
    >
      <Library class="w-5 h-5 text-accent-600 dark:text-accent-400" />
    </div>
    <h1
      class="text-3xl font-display font-bold text-ink-900 dark:text-cream-100"
    >
      My Library
    </h1>
  </div>

  <div
    class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800 animate-fade-in"
  >
    <div class="text-center py-8">
      <Library class="w-12 h-12 text-ink-200 dark:text-ink-700 mx-auto mb-4" />
      <p class="text-ink-600 dark:text-ink-400 text-lg">
        Your personal library is empty.
      </p>
      <p class="text-ink-500 dark:text-ink-500 text-sm mt-1">
        Browse All Books to add some to your collection.
      </p>
    </div>
  </div>
</div>
