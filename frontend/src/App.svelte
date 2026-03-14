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
    class="min-h-screen bg-slate-50 dark:bg-slate-900 flex items-center justify-center"
  >
    <div class="text-center">
      <div
        class="w-16 h-16 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto mb-4"
      ></div>
      <p class="text-slate-600 dark:text-slate-400">Loading...</p>
    </div>
  </div>
{:else if !$user}
  <Auth />
{:else}
  <div class="min-h-screen bg-slate-50 dark:bg-slate-900">
    <Sidebar
      currentView={$currentView}
      onNavigate={(view) => navigate(view)}
      open={sidebarOpen}
      onClose={() => (sidebarOpen = false)}
    />

    <!-- Mobile header with hamburger -->
    <div
      class="sticky top-0 z-30 flex items-center gap-3 bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700 px-4 py-3 md:hidden"
    >
      <button
        onclick={() => (sidebarOpen = true)}
        class="p-1.5 rounded-lg text-slate-600 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
        aria-label="Open menu"
      >
        <Menu class="w-6 h-6" />
      </button>
      <span class="text-lg font-bold text-slate-900 dark:text-white"
        >biblioteka</span
      >
    </div>

    <main class="md:ml-64 p-4 md:p-8">
      <div class="max-w-6xl mx-auto">
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
