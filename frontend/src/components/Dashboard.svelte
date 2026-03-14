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
    <LayoutDashboard class="w-8 h-8 text-blue-600" />
    <h1 class="text-3xl font-bold text-slate-900 dark:text-white">Dashboard</h1>
  </div>

  {#if libraryStore.loaded && libraryStore.libraries.length === 0}
    <div class="bg-white dark:bg-slate-800 rounded-xl p-8 shadow-sm border border-slate-200 dark:border-slate-700">
      <div class="flex items-start gap-4">
        <div class="w-12 h-12 bg-blue-100 dark:bg-blue-900/30 rounded-xl flex items-center justify-center flex-shrink-0">
          <Library class="w-6 h-6 text-blue-600 dark:text-blue-400" />
        </div>
        <div>
          <h2 class="text-lg font-semibold text-slate-900 dark:text-white mb-2">Get started with Biblioteka</h2>
          <p class="text-slate-600 dark:text-slate-400 mb-4">
            To begin managing your books, add a library by pointing it to one or more folders on your system. Biblioteka will organize the books it finds using the Book Per Folder layout.
          </p>
          <button
            onclick={() => routerStore.navigate("libraries/new")}
            class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium"
          >
            <Plus class="w-4 h-4" />
            Add Your First Library
            <ArrowRight class="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-200 dark:border-slate-700">
        <p class="text-sm font-medium text-slate-500 dark:text-slate-400">Total Books</p>
        <p class="text-3xl font-bold text-slate-900 dark:text-white mt-1">0</p>
      </div>
      <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-200 dark:border-slate-700">
        <p class="text-sm font-medium text-slate-500 dark:text-slate-400">Libraries</p>
        <p class="text-3xl font-bold text-slate-900 dark:text-white mt-1">{libraryStore.libraries.length}</p>
      </div>
      <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-200 dark:border-slate-700">
        <p class="text-sm font-medium text-slate-500 dark:text-slate-400">Currently Reading</p>
        <p class="text-3xl font-bold text-slate-900 dark:text-white mt-1">0</p>
      </div>
    </div>

    <div class="mt-8 bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-200 dark:border-slate-700">
      <h2 class="text-lg font-semibold text-slate-900 dark:text-white mb-4">Welcome to Biblioteka</h2>
      <p class="text-slate-600 dark:text-slate-400">
        Your personal book management dashboard. Start by adding books to your library.
      </p>
    </div>
  {/if}
</div>
