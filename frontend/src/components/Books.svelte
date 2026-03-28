<script lang="ts">
  import { BookOpen } from "lucide-svelte";
  import BookList from "./ui/BookList.svelte";
  import { routerStore } from "../stores/router.svelte";
  import * as api from "../lib/api";

  let initialOffset = $derived(
    parseInt(routerStore.queryParams.get("offset") ?? "0", 10) || 0,
  );

  function handlePageChange(offset: number) {
    routerStore.setQueryParam("offset", offset === 0 ? null : String(offset));
  }
</script>

<div>
  <div class="flex items-center gap-3 mb-8">
    <div
      class="w-10 h-10 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center"
    >
      <BookOpen class="w-5 h-5 text-accent-600 dark:text-accent-400" />
    </div>
    <h1
      class="text-3xl font-display font-bold text-ink-900 dark:text-cream-100"
    >
      All Books
    </h1>
  </div>

  <BookList
    fetchBooks={api.listBooks}
    {initialOffset}
    onPageChange={handlePageChange}
  />
</div>
