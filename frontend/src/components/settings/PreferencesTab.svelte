<script lang="ts">
  import { themeStore } from "../../stores/theme.svelte";
  import { Palette } from "lucide-svelte";

  let compactView = $state(false);

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

      <hr class="border-ink-100 dark:border-ink-800" />

      <div>
        <label class="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            bind:checked={compactView}
            class="w-4 h-4 rounded border-ink-300 text-accent-600 outline-none focus:ring-2 focus:ring-accent-500"
          />
          <div>
            <span
              class="block text-sm font-medium text-ink-600 dark:text-ink-300"
            >
              Compact View
            </span>
            <span class="text-xs text-ink-400 dark:text-ink-500">
              Display content in a more compact layout
            </span>
          </div>
        </label>
      </div>
    </div>
  </div>

  <hr class="border-ink-100 dark:border-ink-800" />

  <div>
    <h2
      class="text-lg font-display font-bold text-ink-900 dark:text-cream-100 mb-4"
    >
      About
    </h2>
    <div class="space-y-3 text-sm text-ink-500 dark:text-ink-400">
      <div class="flex justify-between">
        <span>App Version</span>
        <span class="font-medium text-ink-900 dark:text-cream-100">1.0.0</span>
      </div>
      <div class="flex justify-between">
        <span>Last Updated</span>
        <span class="font-medium text-ink-900 dark:text-cream-100"
          >Feb 16, 2026</span
        >
      </div>
    </div>
  </div>
</div>
