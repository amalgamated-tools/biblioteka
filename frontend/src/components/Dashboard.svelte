<script lang="ts">
  import { LayoutDashboard } from "lucide-svelte";
</script>

<div>
  <div class="flex items-center gap-3 mb-8">
    <LayoutDashboard class="w-8 h-8 text-blue-600" />
    <h1 class="text-3xl font-bold text-slate-900 dark:text-white">Dashboard</h1>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
    <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-200 dark:border-slate-700">
      <p class="text-sm font-medium text-slate-500 dark:text-slate-400">Total Books</p>
      <p class="text-3xl font-bold text-slate-900 dark:text-white mt-1">0</p>
    </div>
    <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-200 dark:border-slate-700">
      <p class="text-sm font-medium text-slate-500 dark:text-slate-400">In My Library</p>
      <p class="text-3xl font-bold text-slate-900 dark:text-white mt-1">0</p>
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
</div>
