<script lang="ts">
  import { onMount } from "svelte";
  import { user, authLoading, initAuth } from "./stores/auth";
  import { currentView, navigate } from "./stores/router";
  import { getConfigStatus } from "./lib/api";
  import Auth from "./components/Auth.svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import Movies from "./components/Movies.svelte";
  import TvShows from "./components/TvShows.svelte";
  import Services from "./components/Services.svelte";
  import Settings from "./components/Settings.svelte";

  onMount(async () => {
    await initAuth();

    // If the admin user is logged in and TMDB isn't configured, redirect to settings
    if ($user) {
      try {
        const status = await getConfigStatus();
        if (status.is_admin && !status.tmdb_configured) {
          navigate("settings");
        }
      } catch {
        // ignore - config check is best-effort
      }
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
    <Sidebar
      currentView={$currentView}
      onNavigate={(view) => navigate(view)}
    />

    <main class="ml-64 p-8">
      <div class="max-w-6xl mx-auto">
        {#if $currentView === "movies"}
          <Movies />
        {:else if $currentView === "tvshows"}
          <TvShows />
        {:else if $currentView === "services"}
          <Services />
        {:else if $currentView === "settings"}
          <Settings />
        {/if}
      </div>
    </main>
  </div>
{/if}
