<script lang="ts">
  import { themeStore } from "../../stores/theme.svelte";
  import { Palette } from "lucide-svelte";

  const themes = ["light", "dark", "auto"] as const;
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
>
  <div>
    <h2
      class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
    >
      <Palette class="w-5 h-5 text-accent-600" />
      Display Preferences
    </h2>
    <div class="space-y-6">
      <fieldset>
        <legend
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-3"
        >
          Theme
        </legend>
        <div class="flex gap-3">
          {#each themes as t (t)}
            <button
              type="button"
              aria-pressed={themeStore.preference === t}
              onclick={() => themeStore.set(t)}
              class="px-5 py-2.5 rounded-xl font-medium capitalize transition-all {themeStore.preference ===
              t
                ? 'bg-accent-600 text-white shadow-md shadow-accent-600/20'
                : 'bg-ink-50 text-ink-600 hover:bg-ink-100 dark:bg-ink-800 dark:text-ink-300 dark:hover:bg-ink-700'}"
            >
              {t}
            </button>
          {/each}
        </div>
        <p class="text-xs text-ink-400 dark:text-ink-500 mt-2">
          Choose how you prefer biblioteka to appear
        </p>
      </fieldset>
    </div>
  </div>
</div>
