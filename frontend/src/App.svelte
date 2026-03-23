<script lang="ts">
  import { onMount } from "svelte";
  import { authStore } from "./stores/auth.svelte";
  import { routerStore } from "./stores/router.svelte";
  import { libraryStore } from "./stores/libraries.svelte";
  import Auth from "./components/Auth.svelte";
  import Dashboard from "./components/Dashboard.svelte";
  import Books from "./components/Books.svelte";
  import MyLibrary from "./components/MyLibrary.svelte";
  import Libraries from "./components/Libraries.svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import Settings from "./components/Settings.svelte";
  import { Menu } from "lucide-svelte";

  let sidebarOpen = $state(false);

  onMount(async () => {
    await authStore.init();
  });

  // Redirect away from books view when no libraries exist
  $effect(() => {
    if (
      routerStore.currentView === "books" &&
      libraryStore.loaded &&
      libraryStore.libraries.length === 0
    ) {
      routerStore.navigate("dashboard");
    }
  });

  // Update document title to reflect the current view (WCAG 2.4.2)
  $effect(() => {
    document.title = routerStore.pageTitle;
  });

  // Close the mobile sidebar whenever the active view changes
  $effect(() => {
    void routerStore.currentView;
    sidebarOpen = false;
  });
</script>

{#if authStore.loading}
  <div
    class="min-h-screen bg-cream-50 dark:bg-ink-950 flex items-center justify-center relative bg-texture"
  >
    <div class="text-center animate-fade-in">
      <div class="relative w-16 h-16 mx-auto mb-6">
        <div
          class="absolute inset-0 rounded-2xl bg-accent-500/20 dark:bg-accent-500/10"
          style="animation: spin-slow 3s linear infinite"
        ></div>
        <div
          class="absolute inset-1 rounded-xl bg-accent-600 flex items-center justify-center"
        >
          <span class="text-white font-display text-2xl font-bold">B</span>
        </div>
      </div>
      <p class="text-ink-400 dark:text-ink-300 font-body text-sm tracking-wide">
        Loading your library…
      </p>
    </div>
  </div>
{:else if !authStore.user}
  <Auth />
{:else}
  <div class="min-h-screen bg-cream-50 dark:bg-ink-950 relative bg-texture">
    <a
      href="#main-content"
      onclick={(e: MouseEvent) => {
        e.preventDefault();
        document.getElementById("main-content")?.focus();
      }}
      class="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[100] focus:rounded-xl focus:bg-accent-600 focus:px-4 focus:py-2 focus:font-semibold focus:text-white"
    >
      Skip to main content
    </a>

    <Sidebar
      currentView={routerStore.currentView}
      subPath={routerStore.subPath}
      open={sidebarOpen}
      onClose={() => (sidebarOpen = false)}
    />

    <!-- Mobile header with hamburger -->
    <header
      aria-label="Site header"
      class="sticky top-0 z-30 flex items-center gap-3 bg-cream-50/90 dark:bg-ink-950/90 backdrop-blur-md border-b border-ink-100 dark:border-ink-800 px-4 py-3 md:hidden"
    >
      <button
        onclick={() => (sidebarOpen = true)}
        class="p-1.5 rounded-lg text-ink-500 dark:text-ink-300 hover:bg-ink-100 dark:hover:bg-ink-800 transition-colors"
        aria-label="Open menu"
      >
        <Menu class="w-6 h-6" />
      </button>
      <span
        class="text-lg font-display font-bold text-ink-900 dark:text-cream-100"
        >biblioteka</span
      >
    </div>

    <main id="main-content" tabindex="-1" class="md:ml-64 p-4 md:p-8">
      <div class="max-w-6xl mx-auto animate-fade-in">
        {#if routerStore.currentView === "dashboard"}
          <Dashboard />
        {:else if routerStore.currentView === "books"}
          <Books />
        {:else if routerStore.currentView === "my-library"}
          <MyLibrary />
        {:else if routerStore.currentView === "libraries"}
          <Libraries />
        {:else if routerStore.currentView === "settings"}
          <Settings />
        {/if}
      </div>
    </main>
  </div>
{/if}
