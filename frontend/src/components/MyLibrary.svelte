<script lang="ts">
  import { Library } from "lucide-svelte";
</script>

<div>
  <div class="flex items-center gap-3 mb-8">
    <Library class="w-8 h-8 text-blue-600" />
    <h1 class="text-3xl font-bold text-slate-900 dark:text-white">My Library</h1>
  </div>

  <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-200 dark:border-slate-700">
    <p class="text-slate-600 dark:text-slate-400">
      Your personal library is empty. Browse All Books to add some to your collection.
    </p>
  </div>
</div>
