<script lang="ts">
  import { libraryStore } from "../stores/libraries.svelte";
  import { routerStore } from "../stores/router.svelte";
  import { Plus, Library as LibraryIcon } from "lucide-svelte";
  import AlertBanner from "./ui/AlertBanner.svelte";
  import LibraryView from "./libraries/LibraryView.svelte";
  import LibraryForm from "./libraries/LibraryForm.svelte";
  import FirstLibraryWizard from "./libraries/FirstLibraryWizard.svelte";

  let error: string | null = $state(null);

  // Determine mode from subPath: "new", "edit/{id}", "setup", "{id}" (view), or empty
  let mode: "create" | "edit" | "view" | "setup" | "empty" = $derived.by(() => {
    const sp = routerStore.subPath;
    if (sp === "new") return "create";
    if (sp.startsWith("edit/")) return "edit";
    if (sp === "setup") return "setup";
    if (sp !== "") return "view";
    return "empty";
  });

  let editId: string = $derived.by(() => {
    const sp = routerStore.subPath;
    if (sp.startsWith("edit/")) return sp.slice(5);
    return "";
  });

  let viewId: string = $derived.by(() => {
    const sp = routerStore.subPath;
    if (sp === "new" || sp.startsWith("edit/") || sp === "setup" || sp === "")
      return "";
    return sp;
  });

  let viewLibrary = $derived.by(() => {
    if (!viewId) return null;
    return libraryStore.libraries.find((l) => l.id === viewId) ?? null;
  });

  $effect(() => {
    if (!libraryStore.loaded) {
      void (async () => {
        try {
          await libraryStore.load();
        } catch (e) {
          error = e instanceof Error ? e.message : "Failed to load libraries";
        }
      })();
    }
  });

  // Redirect away from setup wizard if the user already has libraries.
  $effect(() => {
    if (
      mode === "setup" &&
      libraryStore.loaded &&
      libraryStore.libraries.length > 0
    ) {
      routerStore.navigate("libraries");
    }
  });
</script>

<div>
  {#if error && mode !== "view"}
    <AlertBanner variant="error" class="mb-4">{error}</AlertBanner>
  {/if}

  {#if mode === "view"}
    <LibraryView library={viewLibrary} libraryId={viewId} {error} />
  {:else if mode === "create" || mode === "edit"}
    <LibraryForm {mode} {editId} />
  {:else if mode === "setup"}
    <FirstLibraryWizard />
  {:else}
    <div
      class="flex flex-col items-center justify-center py-24 animate-fade-in"
    >
      <h1 class="sr-only">Libraries</h1>
      <LibraryIcon
        class="w-16 h-16 text-ink-200 dark:text-ink-700 mb-6"
        aria-hidden="true"
      />
      {#if libraryStore.libraries.length === 0}
        <button
          onclick={() => routerStore.navigate("libraries/new")}
          class="inline-flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-accent-600 to-accent-700 text-white rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all text-base font-semibold shadow-md shadow-accent-600/20 hover:shadow-lg hover:shadow-accent-600/30 active:scale-[0.98]"
        >
          <Plus class="w-5 h-5" aria-hidden="true" />
          Add A Library
        </button>
      {:else}
        <p class="text-ink-500 dark:text-ink-300">
          Select a library from the sidebar or create a new one.
        </p>
      {/if}
    </div>
  {/if}
</div>
