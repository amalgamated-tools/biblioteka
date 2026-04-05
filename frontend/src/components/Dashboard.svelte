<script lang="ts">
  import { LayoutDashboard, Library, Plus, ArrowRight } from "lucide-svelte";
  import { libraryStore } from "../stores/libraries.svelte";
  import { routerStore } from "../stores/router.svelte";

  $effect(() => {
    if (!libraryStore.loaded) {
      libraryStore.load();
    }
  });
</script>

<div>
  <div class="flex items-center gap-3 mb-8">
    <div
      class="w-10 h-10 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center"
    >
      <LayoutDashboard class="w-5 h-5 text-accent-600 dark:text-accent-400" />
    </div>
    <h1
      class="text-3xl font-display font-bold text-ink-900 dark:text-cream-100"
    >
      Dashboard
    </h1>
  </div>

  {#if libraryStore.loaded && libraryStore.libraries.length === 0}
    <div
      class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800 animate-fade-in-up"
    >
      <div class="flex items-start gap-5">
        <div
          class="w-14 h-14 bg-gradient-to-br from-accent-100 to-accent-200 dark:from-accent-800/30 dark:to-accent-700/20 rounded-2xl flex items-center justify-center flex-shrink-0"
        >
          <Library class="w-7 h-7 text-accent-600 dark:text-accent-400" />
        </div>
        <div>
          <h2
            class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-2"
          >
            Get started with Biblioteka
          </h2>
          <p class="text-ink-400 dark:text-ink-400 mb-5 leading-relaxed">
            To begin managing your books, add a library by pointing it to one or
            more folders on your system. Biblioteka will organize the books it
            finds using the Book Per Folder layout.
          </p>
          <button
            onclick={() => routerStore.navigate("libraries/new")}
            class="inline-flex items-center gap-2 px-5 py-2.5 bg-gradient-to-r from-accent-600 to-accent-700 text-white rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all text-sm font-semibold shadow-md shadow-accent-600/20 hover:shadow-lg hover:shadow-accent-600/30 active:scale-[0.98]"
          >
            <Plus class="w-4 h-4" />
            Add Your First Library
            <ArrowRight class="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-3 gap-5 stagger">
      <div
        class="group bg-white dark:bg-ink-900 rounded-2xl p-6 shadow-sm border border-ink-100 dark:border-ink-800 hover:shadow-md hover:border-accent-200 dark:hover:border-accent-800/30 transition-all"
      >
        <dl class="flex flex-col gap-2">
          <dt class="text-sm font-medium text-ink-400 dark:text-ink-400">
            Total Books
          </dt>
          <dd
            class="text-4xl font-display font-bold text-ink-900 dark:text-cream-100 m-0"
          >
            0
          </dd>
        </dl>
      </div>
      <div
        class="group bg-white dark:bg-ink-900 rounded-2xl p-6 shadow-sm border border-ink-100 dark:border-ink-800 hover:shadow-md hover:border-accent-200 dark:hover:border-accent-800/30 transition-all"
      >
        <dl class="flex flex-col gap-2">
          <dt class="text-sm font-medium text-ink-400 dark:text-ink-400">
            Libraries
          </dt>
          <dd
            class="text-4xl font-display font-bold text-ink-900 dark:text-cream-100 m-0"
          >
            {libraryStore.libraries.length}
          </dd>
        </dl>
      </div>
      <div
        class="group bg-white dark:bg-ink-900 rounded-2xl p-6 shadow-sm border border-ink-100 dark:border-ink-800 hover:shadow-md hover:border-accent-200 dark:hover:border-accent-800/30 transition-all"
      >
        <dl class="flex flex-col gap-2">
          <dt class="text-sm font-medium text-ink-400 dark:text-ink-400">
            Currently Reading
          </dt>
          <dd
            class="text-4xl font-display font-bold text-ink-900 dark:text-cream-100 m-0"
          >
            0
          </dd>
        </dl>
      </div>
    </div>

    <div
      class="mt-8 bg-white dark:bg-ink-900 rounded-2xl p-6 shadow-sm border border-ink-100 dark:border-ink-800 animate-fade-in"
    >
      <h2
        class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-3"
      >
        Welcome to Biblioteka
      </h2>
      <p class="text-ink-400 dark:text-ink-400 leading-relaxed">
        Your personal book management dashboard. Start by adding books to your
        library.
      </p>
    </div>
  {/if}
</div>
