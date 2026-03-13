<script lang="ts">
  import { untrack } from "svelte";
  import { Film, Heart, Search, LayoutGrid, List } from "lucide-svelte";
  import {
    searchMovies,
    listMovies,
    likeMovie,
    unlikeMovie,
    getMovieProviders,
    getUserWatchProviders,
    ApiError,
  } from "../lib/api";
  import type { Movie, MovieSearchResult, MovieProviders } from "../types";
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
  let myMovies: Movie[] = $state([]);

  // Browse state
  let browseResults: MovieSearchResult[] = $state([]);
  let browseLoading = $state(false);
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let togglingTmdbId: number | null = $state(null);

  // Provider state
  let providerCache: Record<number, MovieProviders | null> = $state({});
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

    // Mark all as loading
    for (const id of toFetch) {
      providerCache[id] = null;
    }

    // Process with concurrency limit of 3
    const concurrency = 3;
    let i = 0;
    async function next() {
      while (i < toFetch.length) {
        if (controller.signal.aborted) return;
        const id = toFetch[i++];
        try {
          const data = await getMovieProviders(id);
          if (!controller.signal.aborted) {
            providerCache[id] = data;
          }
        } catch {
          if (!controller.signal.aborted) {
            providerCache[id] = { tmdb_id: id, stream: [], rent: [], buy: [] };
          }
        }
      }
    }
    await Promise.all(Array.from({ length: concurrency }, next));
  }

  async function loadCollection() {
    loading = true;
    try {
      myMovies = await listMovies();
      error = null;
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to load movies";
    } finally {
      loading = false;
    }
  }

  async function loadBrowse() {
    browseLoading = true;
    try {
      browseResults = await searchMovies(query);
      error = null;
      tmdbNotConfigured = false;
    } catch (err) {
      if (err instanceof ApiError && err.status === 503) {
        tmdbNotConfigured = true;
        error = null;
      } else {
        error = err instanceof Error ? err.message : "Failed to search movies";
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
    navigate(newTab === "browse" ? "movies/browse" : "movies");
  }

  async function toggleLike(movie: MovieSearchResult) {
    togglingTmdbId = movie.tmdb_id;
    error = null;
    try {
      if (movie.liked) {
        await unlikeMovie(movie.tmdb_id);
      } else {
        await likeMovie({
          tmdb_id: movie.tmdb_id,
          title: movie.title,
          overview: movie.overview || undefined,
          year: movie.year || undefined,
          poster_url: movie.poster_url || undefined,
        });
      }
      // Refresh both browse results (liked status) and collection count
      await Promise.all([loadBrowse(), loadCollection()]);
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to update";
    } finally {
      togglingTmdbId = null;
    }
  }

  async function unlikeFromCollection(movie: Movie) {
    if (!movie.tmdb_id) return;
    error = null;
    try {
      await unlikeMovie(movie.tmdb_id);
      myMovies = myMovies.filter((m) => m.id !== movie.id);
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to remove movie";
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
    if (tab === "collection" && myMovies.length > 0) {
      const ids = myMovies
        .filter((m) => m.tmdb_id != null)
        .map((m) => m.tmdb_id!);
      untrack(() => batchLoadProviders(ids));
    }
  });
</script>

<div>
  <div class="mb-6">
    <div class="flex items-center justify-between mb-2">
      <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">
        Movies
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
      Browse and like movies to build your collection
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
      {#if myMovies.length > 0}
        <span
          class="ml-1.5 px-2 py-0.5 text-xs rounded-full {tab === 'collection'
            ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400'
            : 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400'}"
        >
          {myMovies.length}
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
      TMDB API key is not configured. Movie search requires a TMDB API key.
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
          class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400 dark:text-slate-500"
        />
        <input
          type="text"
          bind:value={query}
          oninput={handleSearchInput}
          class="w-full pl-10 pr-4 py-3 border border-slate-300 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
          placeholder="Search movies..."
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
        <Film
          class="w-16 h-16 text-slate-300 dark:text-slate-600 mx-auto mb-4"
        />
        <p class="text-slate-600 dark:text-slate-400">No movies found</p>
      </div>
    {:else if viewMode === "grid"}
      <div
        class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4"
      >
        {#each browseResults as movie (movie.tmdb_id)}
          <div
            class="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden group relative"
          >
            {#if movie.poster_url}
              <img
                src={movie.poster_url}
                alt={movie.title}
                class="w-full aspect-[2/3] object-cover"
              />
            {:else}
              <div
                class="w-full aspect-[2/3] bg-slate-100 dark:bg-slate-700 flex items-center justify-center"
              >
                <Film class="w-12 h-12 text-slate-300 dark:text-slate-600" />
              </div>
            {/if}
            <div class="p-3">
              <h3
                class="font-semibold text-sm text-slate-900 dark:text-slate-100 truncate"
              >
                {movie.title}
              </h3>
              <p class="text-xs text-slate-500 dark:text-slate-400">
                {movie.year}
              </p>
            </div>
            <button
              onclick={() => toggleLike(movie)}
              disabled={togglingTmdbId === movie.tmdb_id}
              class="absolute top-2 right-2 p-1.5 rounded-full transition-all {movie.liked
                ? 'bg-red-500 text-white'
                : 'bg-black/40 text-white opacity-0 group-hover:opacity-100'} disabled:opacity-50"
              title={movie.liked
                ? "Remove from collection"
                : "Add to collection"}
            >
              <Heart
                class="w-5 h-5"
                fill={movie.liked ? "currentColor" : "none"}
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
            {#each browseResults as movie (movie.tmdb_id)}
              <tr
                class="border-b border-slate-100 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700"
              >
                <td class="px-4 py-3">
                  {#if movie.poster_url}
                    <img
                      src={movie.poster_url}
                      alt={movie.title}
                      class="w-24 aspect-[2/3] object-cover rounded"
                    />
                  {:else}
                    <div
                      class="w-24 aspect-[2/3] bg-slate-100 dark:bg-slate-700 rounded flex items-center justify-center"
                    >
                      <Film
                        class="w-5 h-5 text-slate-300 dark:text-slate-600"
                      />
                    </div>
                  {/if}
                </td>
                <td class="px-4 py-3">
                  <span class="font-medium text-slate-900 dark:text-slate-100"
                    >{movie.title}</span
                  >
                </td>
                <td class="px-4 py-3 text-sm text-slate-600 dark:text-slate-400"
                  >{movie.year}</td
                >
                <td
                  class="px-4 py-3 text-sm text-slate-500 dark:text-slate-400 max-w-xs truncate"
                  >{movie.overview}</td
                >
                <td class="px-4 py-3">
                  <button
                    onclick={() => toggleLike(movie)}
                    disabled={togglingTmdbId === movie.tmdb_id}
                    class="p-1.5 rounded-lg transition-colors {movie.liked
                      ? 'text-red-500 hover:bg-red-50 dark:hover:bg-red-900/30'
                      : 'text-slate-400 dark:text-slate-500 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/30'} disabled:opacity-50"
                    title={movie.liked
                      ? "Remove from collection"
                      : "Add to collection"}
                  >
                    <Heart
                      class="w-5 h-5"
                      fill={movie.liked ? "currentColor" : "none"}
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
  {:else if myMovies.length === 0}
    <div
      class="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-12 text-center"
    >
      <Film class="w-16 h-16 text-slate-300 dark:text-slate-600 mx-auto mb-4" />
      <h3 class="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-2">
        No movies yet
      </h3>
      <p class="text-slate-600 dark:text-slate-400 mb-6 max-w-md mx-auto">
        Browse the catalog and like movies to add them to your collection.
      </p>
      <button
        onclick={() => switchTab("browse")}
        class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors"
      >
        Browse Movies
      </button>
    </div>
  {:else if viewMode === "grid"}
    <div
      class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4"
    >
      {#each myMovies as movie (movie.id)}
        <div
          class="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden group relative"
        >
          {#if movie.poster_url}
            <img
              src={movie.poster_url}
              alt={movie.title}
              class="w-full aspect-[2/3] object-cover"
            />
          {:else}
            <div
              class="w-full aspect-[2/3] bg-slate-100 dark:bg-slate-700 flex items-center justify-center"
            >
              <Film class="w-12 h-12 text-slate-300 dark:text-slate-600" />
            </div>
          {/if}
          <div class="p-3">
            <h3
              class="font-semibold text-sm text-slate-900 dark:text-slate-100 truncate"
            >
              {movie.title}
            </h3>
            {#if movie.year}
              <p class="text-xs text-slate-500 dark:text-slate-400">
                {movie.year}
              </p>
            {/if}
            {#if movie.tmdb_id}
              {#if providerCache[movie.tmdb_id] === null}
                <div class="flex items-center gap-1 mt-1.5">
                  <div
                    class="w-3 h-3 border-2 border-slate-300 dark:border-slate-600 border-t-transparent rounded-full animate-spin"
                  ></div>
                </div>
              {:else if providerCache[movie.tmdb_id]?.stream.length}
                <div class="flex flex-wrap gap-1 mt-1.5">
                  {#each providerCache[movie.tmdb_id]!.stream as provider (provider.id)}
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
              {:else if providerCache[movie.tmdb_id] !== undefined}
                <p class="text-xs text-slate-400 dark:text-slate-500 mt-1.5">
                  Not streaming
                </p>
              {/if}
            {/if}
          </div>
          <button
            onclick={() => unlikeFromCollection(movie)}
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
          {#each myMovies as movie (movie.id)}
            <tr
              class="border-b border-slate-100 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700 group"
            >
              <td class="px-4 py-3">
                {#if movie.poster_url}
                  <img
                    src={movie.poster_url}
                    alt={movie.title}
                    class="w-24 aspect-[2/3] object-cover rounded"
                  />
                {:else}
                  <div
                    class="w-24 aspect-[2/3] bg-slate-100 dark:bg-slate-700 rounded flex items-center justify-center"
                  >
                    <Film class="w-5 h-5 text-slate-300 dark:text-slate-600" />
                  </div>
                {/if}
              </td>
              <td class="px-4 py-3">
                <span class="font-medium text-slate-900 dark:text-slate-100"
                  >{movie.title}</span
                >
              </td>
              <td class="px-4 py-3 text-sm text-slate-600 dark:text-slate-400"
                >{movie.year ?? "—"}</td
              >
              <td class="px-4 py-3">
                <span
                  class="inline-block px-2 py-1 text-xs font-medium rounded-full bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400"
                >
                  {movie.status}
                </span>
              </td>
              <td class="px-4 py-3">
                {#if movie.tmdb_id}
                  {#if providerCache[movie.tmdb_id] === null}
                    <div
                      class="w-4 h-4 border-2 border-slate-400 dark:border-slate-500 border-t-transparent rounded-full animate-spin"
                    ></div>
                  {:else if providerCache[movie.tmdb_id]?.stream.length}
                    <div class="flex gap-1 flex-wrap">
                      {#each providerCache[movie.tmdb_id]!.stream.slice(0, 4) as provider (provider.id)}
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
                  {:else if providerCache[movie.tmdb_id] !== undefined}
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
                  onclick={() => unlikeFromCollection(movie)}
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
