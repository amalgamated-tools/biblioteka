<script lang="ts">
  import { onMount } from "svelte";
  import { user, authLoading, initAuth } from "./stores/auth";
  import { currentView, navigate } from "./stores/router";
  import { libraries, librariesLoaded } from "./stores/libraries";
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
    await initAuth();
  });

  // Redirect away from books view when no libraries exist
  $effect(() => {
    if ($currentView === "books" && $librariesLoaded && $libraries.length === 0) {
      navigate("dashboard");
    }
  });

  // Close the mobile sidebar whenever the active view changes
  $effect(() => {
    void $currentView;
    sidebarOpen = false;
  });
</script>

{#if $authLoading}
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
      <p class="text-ink-400 dark:text-ink-400 font-body text-sm tracking-wide">Loading your library…</p>
    </div>
  </div>
{:else if !$user}
  <Auth />
{:else}
  <div class="min-h-screen bg-cream-50 dark:bg-ink-950 relative bg-texture">
    <Sidebar
      currentView={$currentView}
      onNavigate={(view) => navigate(view)}
      open={sidebarOpen}
      onClose={() => (sidebarOpen = false)}
    />

    <!-- Mobile header with hamburger -->
    <div
      class="sticky top-0 z-30 flex items-center gap-3 bg-cream-50/90 dark:bg-ink-950/90 backdrop-blur-md border-b border-ink-100 dark:border-ink-800 px-4 py-3 md:hidden"
    >
      <button
        onclick={() => (sidebarOpen = true)}
        class="p-1.5 rounded-lg text-ink-500 dark:text-ink-300 hover:bg-ink-100 dark:hover:bg-ink-800 transition-colors"
        aria-label="Open menu"
      >
        <Menu class="w-6 h-6" />
      </button>
      <span class="text-lg font-display font-bold text-ink-900 dark:text-cream-100"
        >biblioteka</span
      >
    </div>

    <main class="md:ml-64 p-4 md:p-8">
      <div class="max-w-6xl mx-auto animate-fade-in">
        {#if $currentView === "dashboard"}
          <Dashboard />
        {:else if $currentView === "books"}
          <Books />
        {:else if $currentView === "my-library"}
          <MyLibrary />
        {:else if $currentView === "libraries"}
          <Libraries />
        {:else if $currentView === "settings"}
          <Settings />
        {/if}
      </div>
    </main>
  </div>
{/if}
