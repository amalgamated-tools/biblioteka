<script lang="ts">
  import { untrack } from "svelte";
  import { Tv, Heart, Search, LayoutGrid, List } from "lucide-svelte";
  import {
    searchTvSeries,
    listTvSeries,
    likeTvSeries,
    unlikeTvSeries,
    getTvSeriesProviders,
    getUserWatchProviders,
    ApiError,
  } from "../lib/api";
  import type {
    TvSeries,
    TvSeriesSearchResult,
    TvSeriesProviders,
  } from "../types";
  import { subPath, navigate } from "../stores/router";

  type Tab = "collection" | "browse";

  let tab: Tab = $derived($subPath === "browse" ? "browse" : "collection");
  let viewMode: "grid" | "table" = $state(
    (localStorage.getItem("biblioteka_view_mode") as "grid" | "table") || "grid",
  );

  function setViewMode(mode: "grid" | "table") {
    viewMode = mode;
    localStorage.setItem("biblioteka_view_mode", mode);
  }

  let query = $state("");
  let loading = $state(true);
  let error: string | null = $state(null);
  let tmdbNotConfigured = $state(false);

  // Collection state
  let mySeries: TvSeries[] = $state([]);

  // Browse state
  let browseResults: TvSeriesSearchResult[] = $state([]);
  let browseLoading = $state(false);
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let togglingTmdbId: number | null = $state(null);

  // Provider state
  let providerCache: Record<number, TvSeriesProviders | null> = $state({});
  let activeBatchController: AbortController | null = null;

  // User's subscribed streaming providers
  let userProviderIds: Set<number> = $state(new Set());

  async function loadUserProviders() {
    try {
      const providers = await getUserWatchProviders();
      userProviderIds = new Set(providers.map((p) => p.provider_id));
    } catch {
      userProviderIds = new Set();
    }
  }

  function isUserProvider(providerId: number): boolean {
    if (userProviderIds.size === 0) return true;
    return userProviderIds.has(providerId);
  }

  async function batchLoadProviders(tmdbIds: number[]) {
    if (activeBatchController) activeBatchController.abort();
    const controller = new AbortController();
    activeBatchController = controller;

    const toFetch = tmdbIds.filter((id) => providerCache[id] == null);
    if (toFetch.length === 0) return;

    for (const id of toFetch) {
      providerCache[id] = null;
    }

    const concurrency = 3;
    let i = 0;
    async function next() {
      while (i < toFetch.length) {
        if (controller.signal.aborted) return;
        const id = toFetch[i++];
        try {
          const data = await getTvSeriesProviders(id);
          if (!controller.signal.aborted) {
            providerCache[id] = data;
          }
        } catch {
          if (!controller.signal.aborted) {
            providerCache[id] = { tmdb_id: id, stream: [], buy: [] };
          }
        }
      }
    }
    await Promise.all(Array.from({ length: concurrency }, next));
  }

  async function loadCollection() {
    loading = true;
    try {
      mySeries = await listTvSeries();
      error = null;
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to load TV shows";
    } finally {
      loading = false;
    }
  }

  async function loadBrowse() {
    browseLoading = true;
    try {
      browseResults = await searchTvSeries(query);
      error = null;
      tmdbNotConfigured = false;
    } catch (err) {
      if (err instanceof ApiError && err.status === 503) {
        tmdbNotConfigured = true;
        error = null;
      } else {
        error =
          err instanceof Error ? err.message : "Failed to search TV shows";
      }
    } finally {
      browseLoading = false;
    }
  }

  function handleSearchInput() {
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => loadBrowse(), 300);
  }

  function switchTab(newTab: Tab) {
    navigate(newTab === "browse" ? "tvshows/browse" : "tvshows");
  }

  async function toggleLike(series: TvSeriesSearchResult) {
    togglingTmdbId = series.tmdb_id;
    error = null;
    try {
      if (series.liked) {
        await unlikeTvSeries(series.tmdb_id);
      } else {
        await likeTvSeries({
          tmdb_id: series.tmdb_id,
          title: series.title,
          overview: series.overview || undefined,
          year: series.year || undefined,
          poster_url: series.poster_url || undefined,
        });
      }
      await Promise.all([loadBrowse(), loadCollection()]);
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to update";
    } finally {
      togglingTmdbId = null;
    }
  }

  async function unlikeFromCollection(series: TvSeries) {
    if (!series.tmdb_id) return;
    error = null;
    try {
      await unlikeTvSeries(series.tmdb_id);
      mySeries = mySeries.filter((s) => s.id !== series.id);
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to remove TV show";
    }
  }

  // Reactively load data when the tab changes (including browser back/forward)
  $effect(() => {
    if (tab === "browse") {
      untrack(() => loadBrowse());
    } else {
      loadCollection();
      loadUserProviders();
    }
  });

  // Batch-fetch providers for collection
  $effect(() => {
    if (tab === "collection" && mySeries.length > 0) {
      const ids = mySeries
        .filter((s) => s.tmdb_id != null)
        .map((s) => s.tmdb_id!);
      untrack(() => batchLoadProviders(ids));
    }
  });
</script>

<div>
  <div class="mb-6">
    <div class="flex items-center justify-between mb-2">
      <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">
        TV Shows
      </h1>
      <div class="flex items-center gap-2">
        <div
          class="flex items-center border border-slate-300 dark:border-slate-600 rounded-lg overflow-hidden"
        >
          <button
            onclick={() => setViewMode("grid")}
            class="p-2 transition-colors {viewMode === 'grid'
              ? 'bg-blue-600 text-white'
              : 'bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700'}"
            title="Grid view"
          >
            <LayoutGrid class="w-5 h-5" />
          </button>
          <button
            onclick={() => setViewMode("table")}
            class="p-2 transition-colors {viewMode === 'table'
              ? 'bg-blue-600 text-white'
              : 'bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700'}"
            title="Table view"
          >
            <List class="w-5 h-5" />
          </button>
        </div>
      </div>
    </div>
    <p class="text-sm text-slate-500 dark:text-slate-400">
      Browse and like TV shows to build your collection
    </p>
  </div>

  <!-- Tabs -->
  <div
    class="flex items-center gap-6 border-b border-slate-200 dark:border-slate-700 mb-6"
  >
    <button
      onclick={() => switchTab("collection")}
      class="pb-3 text-sm font-medium transition-colors border-b-2 {tab ===
      'collection'
        ? 'border-blue-600 text-blue-600'
        : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'}"
    >
      My Collection
      {#if mySeries.length > 0}
        <span
          class="ml-1.5 px-2 py-0.5 text-xs rounded-full {tab === 'collection'
            ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400'
            : 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400'}"
        >
          {mySeries.length}
        </span>
      {/if}
    </button>
    <button
      onclick={() => switchTab("browse")}
      class="pb-3 text-sm font-medium transition-colors border-b-2 {tab ===
      'browse'
        ? 'border-blue-600 text-blue-600'
        : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'}"
    >
      Browse All
    </button>
  </div>

  {#if tmdbNotConfigured}
    <div
      class="bg-amber-50 border border-amber-200 text-amber-800 dark:bg-amber-900/30 dark:border-amber-800 dark:text-amber-400 px-4 py-3 rounded-lg text-sm mb-4"
    >
      TMDB API key is not configured. TV show search requires a TMDB API key.
      <button
        onclick={() => navigate("settings")}
        class="underline font-medium hover:text-amber-900 dark:hover:text-amber-300"
      >
        Configure it in Settings
      </button>
    </div>
  {/if}

  {#if error}
    <div
      class="bg-red-50 border border-red-200 text-red-700 dark:bg-red-900/30 dark:border-red-800 dark:text-red-400 px-4 py-3 rounded-lg text-sm mb-4"
    >
      {error}
    </div>
  {/if}

  <!-- Browse tab -->
  {#if tab === "browse"}
    <div class="mb-6">
      <div class="relative">
        <Search
          class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400"
        />
        <input
          type="text"
          bind:value={query}
          oninput={handleSearchInput}
          class="w-full pl-10 pr-4 py-3 border border-slate-300 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
          placeholder="Search TV shows..."
        />
      </div>
    </div>

    {#if browseLoading}
      <div class="flex items-center justify-center py-12">
        <div
          class="w-12 h-12 border-4 border-blue-600 border-t-transparent rounded-full animate-spin"
        ></div>
      </div>
    {:else if browseResults.length === 0}
      <div
        class="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-12 text-center"
      >
        <Tv class="w-16 h-16 text-slate-300 dark:text-slate-600 mx-auto mb-4" />
        <p class="text-slate-600 dark:text-slate-400">No TV shows found</p>
      </div>
    {:else if viewMode === "grid"}
      <div
        class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4"
      >
        {#each browseResults as series (series.tmdb_id)}
          <div
            class="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden group relative"
          >
            {#if series.poster_url}
              <img
                src={series.poster_url}
                alt={series.title}
                class="w-full aspect-[2/3] object-cover"
              />
            {:else}
              <div
                class="w-full aspect-[2/3] bg-slate-100 dark:bg-slate-700 flex items-center justify-center"
              >
                <Tv class="w-12 h-12 text-slate-300 dark:text-slate-600" />
              </div>
            {/if}
            <div class="p-3">
              <h3
                class="font-semibold text-sm text-slate-900 dark:text-slate-100 truncate"
              >
                {series.title}
              </h3>
              <p class="text-xs text-slate-500 dark:text-slate-400">
                {series.year}
              </p>
            </div>
            <button
              onclick={() => toggleLike(series)}
              disabled={togglingTmdbId === series.tmdb_id}
              class="absolute top-2 right-2 p-1.5 rounded-full transition-all {series.liked
                ? 'bg-red-500 text-white'
                : 'bg-black/40 text-white opacity-0 group-hover:opacity-100'} disabled:opacity-50"
              title={series.liked
                ? "Remove from collection"
                : "Add to collection"}
            >
              <Heart
                class="w-5 h-5"
                fill={series.liked ? "currentColor" : "none"}
              />
            </button>
          </div>
        {/each}
      </div>
    {:else}
      <div
        class="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden"
      >
        <table class="w-full">
          <thead>
            <tr
              class="border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 text-left"
            >
              <th
                class="px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider w-12"
              ></th>
              <th
                class="px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider"
                >Title</th
              >
              <th
                class="px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider w-20"
                >Year</th
              >
              <th
                class="px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider"
                >Overview</th
              >
              <th class="px-4 py-3 w-12"></th>
            </tr>
          </thead>
          <tbody>
            {#each browseResults as series (series.tmdb_id)}
              <tr
                class="border-b border-slate-100 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700"
              >
                <td class="px-4 py-3">
                  {#if series.poster_url}
                    <img
                      src={series.poster_url}
                      alt={series.title}
                      class="w-24 aspect-[2/3] object-cover rounded"
                    />
                  {:else}
                    <div
                      class="w-24 aspect-[2/3] bg-slate-100 dark:bg-slate-700 rounded flex items-center justify-center"
                    >
                      <Tv class="w-5 h-5 text-slate-300 dark:text-slate-600" />
                    </div>
                  {/if}
                </td>
                <td class="px-4 py-3">
                  <span class="font-medium text-slate-900 dark:text-slate-100"
                    >{series.title}</span
                  >
                </td>
                <td class="px-4 py-3 text-sm text-slate-600 dark:text-slate-400"
                  >{series.year}</td
                >
                <td
                  class="px-4 py-3 text-sm text-slate-500 dark:text-slate-400 max-w-xs truncate"
                  >{series.overview}</td
                >
                <td class="px-4 py-3">
                  <button
                    onclick={() => toggleLike(series)}
                    disabled={togglingTmdbId === series.tmdb_id}
                    class="p-1.5 rounded-lg transition-colors {series.liked
                      ? 'text-red-500 hover:bg-red-50 dark:hover:bg-red-900/30'
                      : 'text-slate-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/30'} disabled:opacity-50"
                    title={series.liked
                      ? "Remove from collection"
                      : "Add to collection"}
                  >
                    <Heart
                      class="w-5 h-5"
                      fill={series.liked ? "currentColor" : "none"}
                    />
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}

    <!-- Collection tab -->
  {:else if loading}
    <div class="flex items-center justify-center py-12">
      <div
        class="w-12 h-12 border-4 border-blue-600 border-t-transparent rounded-full animate-spin"
      ></div>
    </div>
  {:else if mySeries.length === 0}
    <div
      class="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-12 text-center"
    >
      <Tv class="w-16 h-16 text-slate-300 dark:text-slate-600 mx-auto mb-4" />
      <h3 class="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-2">
        No TV shows yet
      </h3>
      <p class="text-slate-600 dark:text-slate-400 mb-6 max-w-md mx-auto">
        Browse the catalog and like TV shows to add them to your collection.
      </p>
      <button
        onclick={() => switchTab("browse")}
        class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors"
      >
        Browse TV Shows
      </button>
    </div>
  {:else if viewMode === "grid"}
    <div
      class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4"
    >
      {#each mySeries as series (series.id)}
        <div
          class="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden group relative"
        >
          {#if series.poster_url}
            <img
              src={series.poster_url}
              alt={series.title}
              class="w-full aspect-[2/3] object-cover"
            />
          {:else}
            <div
              class="w-full aspect-[2/3] bg-slate-100 dark:bg-slate-700 flex items-center justify-center"
            >
              <Tv class="w-12 h-12 text-slate-300 dark:text-slate-600" />
            </div>
          {/if}
          <div class="p-3">
            <h3
              class="font-semibold text-sm text-slate-900 dark:text-slate-100 truncate"
            >
              {series.title}
            </h3>
            {#if series.year}
              <p class="text-xs text-slate-500 dark:text-slate-400">
                {series.year}
              </p>
            {/if}
            {#if series.tmdb_id}
              {#if providerCache[series.tmdb_id] === null}
                <div class="flex items-center gap-1 mt-1.5">
                  <div
                    class="w-3 h-3 border-2 border-slate-300 dark:border-slate-600 border-t-transparent rounded-full animate-spin"
                  ></div>
                </div>
              {:else if providerCache[series.tmdb_id]?.stream.length}
                <div class="flex flex-wrap gap-1 mt-1.5">
                  {#each providerCache[series.tmdb_id]!.stream as provider (provider.id)}
                    <img
                      src={provider.logo_url}
                      alt={provider.name}
                      title={provider.name}
                      class="w-5 h-5 rounded transition-[filter] duration-200"
                      style={isUserProvider(provider.id)
                        ? ""
                        : "filter: grayscale(1); opacity: 0.5;"}
                    />
                  {/each}
                </div>
              {:else if providerCache[series.tmdb_id] !== undefined}
                <p class="text-xs text-slate-400 dark:text-slate-500 mt-1.5">
                  Not streaming
                </p>
              {/if}
            {/if}
          </div>
          <button
            onclick={() => unlikeFromCollection(series)}
            class="absolute top-2 right-2 p-1.5 rounded-full bg-red-500 text-white opacity-0 group-hover:opacity-100 transition-opacity"
            title="Remove from collection"
          >
            <Heart class="w-5 h-5" fill="currentColor" />
          </button>
        </div>
      {/each}
    </div>
  {:else}
    <div
      class="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden"
    >
      <table class="w-full">
        <thead>
          <tr
            class="border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 text-left"
          >
            <th
              class="px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider w-12"
            ></th>
            <th
              class="px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider"
              >Title</th
            >
            <th
              class="px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider w-20"
              >Year</th
            >
            <th
              class="px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider w-24"
              >Status</th
            >
            <th
              class="px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider w-32"
              >Streaming</th
            >
            <th class="px-4 py-3 w-12"></th>
          </tr>
        </thead>
        <tbody>
          {#each mySeries as series (series.id)}
            <tr
              class="border-b border-slate-100 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700 group"
            >
              <td class="px-4 py-3">
                {#if series.poster_url}
                  <img
                    src={series.poster_url}
                    alt={series.title}
                    class="w-24 aspect-[2/3] object-cover rounded"
                  />
                {:else}
                  <div
                    class="w-24 aspect-[2/3] bg-slate-100 dark:bg-slate-700 rounded flex items-center justify-center"
                  >
                    <Tv class="w-5 h-5 text-slate-300 dark:text-slate-600" />
                  </div>
                {/if}
              </td>
              <td class="px-4 py-3">
                <span class="font-medium text-slate-900 dark:text-slate-100"
                  >{series.title}</span
                >
              </td>
              <td class="px-4 py-3 text-sm text-slate-600 dark:text-slate-400"
                >{series.year ?? "—"}</td
              >
              <td class="px-4 py-3">
                <span
                  class="inline-block px-2 py-1 text-xs font-medium rounded-full bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400"
                >
                  {series.status}
                </span>
              </td>
              <td class="px-4 py-3">
                {#if series.tmdb_id}
                  {#if providerCache[series.tmdb_id] === null}
                    <div
                      class="w-4 h-4 border-2 border-slate-400 dark:border-slate-500 border-t-transparent rounded-full animate-spin"
                    ></div>
                  {:else if providerCache[series.tmdb_id]?.stream.length}
                    <div class="flex gap-1 flex-wrap">
                      {#each providerCache[series.tmdb_id]!.stream.slice(0, 4) as provider (provider.id)}
                        <img
                          src={provider.logo_url}
                          alt={provider.name}
                          title={provider.name}
                          class="w-6 h-6 rounded transition-[filter] duration-200"
                          style={isUserProvider(provider.id)
                            ? ""
                            : "filter: grayscale(1); opacity: 0.5;"}
                        />
                      {/each}
                    </div>
                  {:else if providerCache[series.tmdb_id] !== undefined}
                    <span class="text-xs text-slate-400 dark:text-slate-500"
                      >&mdash;</span
                    >
                  {:else}
                    <span class="text-xs text-slate-400 dark:text-slate-500"
                      >...</span
                    >
                  {/if}
                {:else}
                  <span class="text-xs text-slate-400 dark:text-slate-500"
                    >&mdash;</span
                  >
                {/if}
              </td>
              <td class="px-4 py-3">
                <button
                  onclick={() => unlikeFromCollection(series)}
                  class="p-1.5 text-red-500 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/30 transition-colors opacity-0 group-hover:opacity-100"
                  title="Remove from collection"
                >
                  <Heart class="w-5 h-5" fill="currentColor" />
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
