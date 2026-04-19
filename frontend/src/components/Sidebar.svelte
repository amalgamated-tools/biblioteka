<script lang="ts">
  import { authStore } from "../stores/auth.svelte";
  import type { AppView } from "../stores/router.svelte";
  import { libraryStore } from "../stores/libraries.svelte";
  import { getVersion } from "../lib/api";
  import {
    LayoutDashboard,
    BookOpen,
    Library,
    Plus,
    Settings as SettingsIcon,
    LogOut,
    BookCheck,
    Settings2,
    BookMarked,
    Users,
    Tag,
  } from "lucide-svelte";

  interface Props {
    currentView?: AppView;
    subPath?: string;
    open: boolean;
    onClose: () => void;
  }

  let { currentView, subPath = "", open, onClose }: Props = $props();
  let version = $state("");
  let innerWidth = $state<number>(
    typeof window !== "undefined" ? window.innerWidth : 0,
  );

  let hideSidebar = $derived(!open && innerWidth < 768);

  $effect(() => {
    if (authStore.user && !libraryStore.loaded) {
      libraryStore.load();
    }
  });

  $effect(() => {
    void getVersion()
      .then((v) => {
        version = v;
      })
      .catch(() => {
        // version stays blank; suppress unhandled-rejection noise
      });
  });

  async function handleLogout() {
    await authStore.signOut();
  }
</script>

<svelte:window
  bind:innerWidth
  onkeydown={(e) => {
    if (open && e.key === "Escape") onClose();
  }}
/>

<!-- Mobile overlay backdrop -->
{#if open}
  <button
    class="fixed inset-0 z-40 bg-ink-900/60 dark:bg-ink-950/70 backdrop-blur-sm md:hidden"
    onclick={onClose}
    aria-label="Close sidebar"
  ></button>
{/if}

<aside
  id="main-sidebar"
  aria-label="Main menu"
  aria-hidden={hideSidebar || undefined}
  inert={hideSidebar}
  class="fixed inset-y-0 left-0 z-50 w-64 bg-ink-950 text-white flex flex-col transition-transform duration-200 ease-in-out {open
    ? 'translate-x-0'
    : '-translate-x-full'} md:translate-x-0"
>
  <div class="px-5 py-5 border-b border-ink-800/60">
    <div class="flex items-center gap-3">
      <div
        class="w-10 h-10 bg-gradient-to-br from-accent-500 to-accent-700 rounded-xl flex items-center justify-center shadow-lg shadow-accent-700/20"
      >
        <BookCheck class="w-5 h-5 text-white" aria-hidden="true" />
      </div>
      <div>
        <p class="text-lg font-display font-bold tracking-tight">biblioteka</p>
        <p class="text-xs text-ink-300 truncate">
          {authStore.user?.name || authStore.user?.email}
        </p>
      </div>
    </div>
  </div>

  <nav
    aria-label="Primary navigation"
    class="flex-1 px-3 py-4 space-y-6 overflow-y-auto"
  >
    <!-- Home group -->
    <div role="group" aria-labelledby="sidebar-home-heading">
      <h2
        id="sidebar-home-heading"
        class="px-3 mb-2 text-[10px] font-semibold uppercase tracking-[0.15em] text-ink-300"
      >
        Home
      </h2>
      <div class="space-y-0.5">
        <a
          href="#dashboard"
          aria-current={currentView === "dashboard" ? "page" : undefined}
          class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl font-medium text-sm transition-all {currentView ===
          'dashboard'
            ? 'bg-accent-600 text-white shadow-md shadow-accent-700/30'
            : 'text-ink-300 hover:bg-ink-800/70 hover:text-white'}"
          onclick={onClose}
        >
          <LayoutDashboard class="w-5 h-5" aria-hidden="true" />
          Dashboard
        </a>
        {#if libraryStore.libraries.length > 0}
          <a
            href="#books"
            aria-current={currentView === "books" ? "page" : undefined}
            class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl font-medium text-sm transition-all {currentView ===
            'books'
              ? 'bg-accent-600 text-white shadow-md shadow-accent-700/30'
              : 'text-ink-300 hover:bg-ink-800/70 hover:text-white'}"
            onclick={onClose}
          >
            <BookOpen class="w-5 h-5" aria-hidden="true" />
            All Books
          </a>
        {/if}
        <a
          href="#reading-lists"
          aria-current={currentView === "reading-lists" ? "page" : undefined}
          class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl font-medium text-sm transition-all {currentView ===
          'reading-lists'
            ? 'bg-accent-600 text-white shadow-md shadow-accent-700/30'
            : 'text-ink-300 hover:bg-ink-800/70 hover:text-white'}"
          onclick={onClose}
        >
          <BookMarked class="w-5 h-5" aria-hidden="true" />
          Reading Lists
        </a>
        <a
          href="#groups"
          aria-current={currentView === "groups" ? "page" : undefined}
          class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl font-medium text-sm transition-all {currentView ===
          'groups'
            ? 'bg-accent-600 text-white shadow-md shadow-accent-700/30'
            : 'text-ink-300 hover:bg-ink-800/70 hover:text-white'}"
          onclick={onClose}
        >
          <Users class="w-5 h-5" aria-hidden="true" />
          Reading Groups
        </a>
        <a
          href="#tags"
          aria-current={currentView === "tags" ? "page" : undefined}
          class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl font-medium text-sm transition-all {currentView ===
          'tags'
            ? 'bg-accent-600 text-white shadow-md shadow-accent-700/30'
            : 'text-ink-300 hover:bg-ink-800/70 hover:text-white'}"
          onclick={onClose}
        >
          <Tag class="w-5 h-5" aria-hidden="true" />
          Tags
        </a>
      </div>
    </div>

    <!-- Libraries group -->
    <div role="group" aria-labelledby="sidebar-libraries-heading">
      <div class="flex items-center justify-between px-3 mb-2">
        <h2
          id="sidebar-libraries-heading"
          class="text-[10px] font-semibold uppercase tracking-[0.15em] text-ink-300"
        >
          Libraries
        </h2>
        <a
          href="#libraries/new"
          class="text-ink-500 hover:text-accent-400 transition-colors"
          title="Create library"
          aria-label="Create library"
          onclick={onClose}
        >
          <Plus class="w-4 h-4" aria-hidden="true" />
        </a>
      </div>
      <div class="space-y-0.5">
        {#each libraryStore.libraries as lib (lib.id)}
          <div
            class="group w-full flex items-center gap-3 px-3 py-2 rounded-xl font-medium text-sm transition-all text-ink-300 hover:bg-ink-800/70 hover:text-white"
          >
            <a
              href={`#libraries/${lib.id}`}
              aria-current={currentView === "libraries" &&
              subPath === String(lib.id)
                ? "page"
                : undefined}
              class="flex items-center gap-3 flex-1 min-w-0"
              onclick={onClose}
            >
              <Library
                class="w-4 h-4 flex-shrink-0 text-ink-500 group-hover:text-accent-400 transition-colors"
                aria-hidden="true"
              />
              <span class="truncate flex-1 text-left">{lib.name}</span>
            </a>
            <a
              href={`#libraries/edit/${lib.id}`}
              class="text-ink-600 group-hover:text-accent-400 group-focus-within:text-accent-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-400 rounded transition-all p-0.5 flex-shrink-0"
              aria-label={`Library settings for ${lib.name}`}
              onclick={onClose}
            >
              <Settings2 class="w-3.5 h-3.5" aria-hidden="true" />
            </a>
          </div>
        {/each}
      </div>
    </div>

    <!-- Settings -->
    <div>
      <div class="space-y-0.5">
        <a
          href="#settings"
          aria-current={currentView === "settings" ? "page" : undefined}
          class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl font-medium text-sm transition-all {currentView ===
          'settings'
            ? 'bg-accent-600 text-white shadow-md shadow-accent-700/30'
            : 'text-ink-300 hover:bg-ink-800/70 hover:text-white'}"
          onclick={onClose}
        >
          <SettingsIcon class="w-5 h-5" aria-hidden="true" />
          Settings
        </a>
      </div>
    </div>
  </nav>
  <div class="px-5 py-2 border-t border-ink-800/60">
    <p class="text-[10px] text-ink-300 text-center tracking-wider uppercase">
      {version ? `v${version}` : ""}
    </p>
  </div>

  <div class="px-3 py-4 border-t border-ink-800/60">
    <button
      id="logout-button"
      onclick={handleLogout}
      class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl font-medium text-sm text-ink-300 hover:bg-ink-800/70 hover:text-white transition-all"
    >
      <LogOut class="w-5 h-5" aria-hidden="true" />
      Logout
    </button>
  </div>
</aside>
