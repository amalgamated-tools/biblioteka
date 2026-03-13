<script lang="ts">
  import { BookOpen } from "lucide-svelte";
</script>

<div>
  <div class="flex items-center gap-3 mb-8">
    <BookOpen class="w-8 h-8 text-blue-600" />
    <h1 class="text-3xl font-bold text-slate-900 dark:text-white">All Books</h1>
  </div>

  <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-200 dark:border-slate-700">
    <p class="text-slate-600 dark:text-slate-400">
      No books yet. Books will appear here once they are added.
    </p>
  </div>
</div>
