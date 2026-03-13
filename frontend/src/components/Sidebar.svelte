<script lang="ts">
  import { user, signOut } from "../stores/auth";
  import type { AppView } from "../stores/router";
  import { Library, Settings, LogOut, BookCheck } from "lucide-svelte";

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

  const navItems: { view: AppView; label: string; icon: typeof Library }[] = [
    { view: "libraries", label: "Libraries", icon: Library },
    { view: "settings", label: "Settings", icon: Settings },
  ];
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

  <nav class="flex-1 px-3 py-4 space-y-1">
    {#each navItems as item (item.view)}
      <button
        onclick={() => onNavigate(item.view)}
        class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg font-medium text-sm transition-colors {currentView ===
        item.view
          ? 'bg-blue-600 text-white'
          : 'text-slate-300 hover:bg-slate-800 hover:text-white'}"
      >
        <item.icon class="w-5 h-5" />
        {item.label}
      </button>
    {/each}
  </nav>

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
