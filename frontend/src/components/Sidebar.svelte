<script lang="ts">
  import { user, signOut } from "../stores/auth";
  import type { AppView } from "../stores/router";
  import {
    LayoutDashboard,
    BookOpen,
    Library,
    Plus,
    Settings,
    LogOut,
    BookCheck,
  } from "lucide-svelte";

  interface Props {
    currentView: AppView;
    onNavigate: (view: AppView) => void;
  }

  let { currentView, onNavigate }: Props = $props();

  async function handleLogout() {
    if (confirm("Are you sure you want to logout?")) {
      await signOut();
    }
  }
</script>

<aside
  class="fixed inset-y-0 left-0 z-50 w-64 bg-slate-900 text-white flex flex-col"
>
  <div class="px-5 py-5 border-b border-slate-700">
    <div class="flex items-center gap-3">
      <div
        class="w-10 h-10 bg-blue-600 rounded-xl flex items-center justify-center"
      >
        <BookCheck class="w-6 h-6 text-white" />
      </div>
      <div>
        <h1 class="text-lg font-bold">biblioteka</h1>
        <p class="text-xs text-slate-400 truncate">{$user?.email}</p>
      </div>
    </div>
  </div>

  <nav class="flex-1 px-3 py-4 space-y-6 overflow-y-auto">
    <!-- Home group -->
    <div>
      <p
        class="px-3 mb-1 text-xs font-semibold uppercase tracking-wider text-slate-500"
      >
        Home
      </p>
      <div class="space-y-1">
        <button
          onclick={() => onNavigate("dashboard")}
          class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg font-medium text-sm transition-colors {currentView ===
          'dashboard'
            ? 'bg-blue-600 text-white'
            : 'text-slate-300 hover:bg-slate-800 hover:text-white'}"
        >
          <LayoutDashboard class="w-5 h-5" />
          Dashboard
        </button>
        <button
          onclick={() => onNavigate("books")}
          class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg font-medium text-sm transition-colors {currentView ===
          'books'
            ? 'bg-blue-600 text-white'
            : 'text-slate-300 hover:bg-slate-800 hover:text-white'}"
        >
          <BookOpen class="w-5 h-5" />
          All Books
        </button>
      </div>
    </div>

    <!-- Libraries group -->
    <div>
      <div class="flex items-center justify-between px-3 mb-1">
        <p
          class="text-xs font-semibold uppercase tracking-wider text-slate-500"
        >
          Libraries
        </p>
        <button
          class="text-slate-500 hover:text-slate-300 transition-colors"
          title="Create library"
        >
          <Plus class="w-4 h-4" />
        </button>
      </div>
      <div class="space-y-1">
        <button
          onclick={() => onNavigate("my-library")}
          class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg font-medium text-sm transition-colors {currentView ===
          'my-library'
            ? 'bg-blue-600 text-white'
            : 'text-slate-300 hover:bg-slate-800 hover:text-white'}"
        >
          <Library class="w-5 h-5" />
          My Library
        </button>
      </div>
    </div>

    <!-- Settings -->
    <div>
      <div class="space-y-1">
        <button
          onclick={() => onNavigate("settings")}
          class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg font-medium text-sm transition-colors {currentView ===
          'settings'
            ? 'bg-blue-600 text-white'
            : 'text-slate-300 hover:bg-slate-800 hover:text-white'}"
        >
          <Settings class="w-5 h-5" />
          Settings
        </button>
      </div>
    </div>
  </nav>
  <div class="px-5 py-2 border-t border-slate-700">
    <p class="text-xs text-slate-500 text-center">v0.0.1</p>
  </div>

  <div class="px-3 py-4 border-t border-slate-700">
    <button
      onclick={handleLogout}
      class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg font-medium text-sm text-slate-300 hover:bg-slate-800 hover:text-white transition-colors"
    >
      <LogOut class="w-5 h-5" />
      Logout
    </button>
  </div>
</aside>
