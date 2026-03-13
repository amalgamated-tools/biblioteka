<script lang="ts">
  import { onMount } from "svelte";
  import { user, authLoading, initAuth } from "./stores/auth";
  import { currentView, navigate } from "./stores/router";
  import { libraries } from "./stores/libraries";
  import Auth from "./components/Auth.svelte";
  import Dashboard from "./components/Dashboard.svelte";
  import Books from "./components/Books.svelte";
  import MyLibrary from "./components/MyLibrary.svelte";
  import Libraries from "./components/Libraries.svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import Settings from "./components/Settings.svelte";

  onMount(async () => {
    await initAuth();
  });

  // Redirect away from books view when no libraries exist
  $effect(() => {
    if ($currentView === "books" && $libraries.length === 0) {
      navigate("dashboard");
    }
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
    <Sidebar currentView={$currentView} onNavigate={(view) => navigate(view)} />

    <main class="ml-64 p-8">
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
