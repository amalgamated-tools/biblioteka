<script lang="ts">
  import { onMount } from "svelte";
  import { Server, Plus, Trash2, Edit2, ExternalLink } from "lucide-svelte";
  import {
    listArrServices,
    createArrService,
    updateArrService,
    deleteArrService,
  } from "../lib/api";
  import { user } from "../stores/auth";
  import ArrServiceModal from "./ArrServiceModal.svelte";
  import type { ArrService, ArrServiceInput } from "../types";

  let arrServices: ArrService[] = $state([]);
  let loading = $state(true);
  let error: string | null = $state(null);
  let showModal = $state(false);
  let editingService: ArrService | null = $state(null);
  let isAdmin = $derived($user?.is_admin ?? false);

  const typeLabels: Record<string, string> = {
    radarr: "Radarr",
    sonarr: "Sonarr",
    prowlarr: "Prowlarr",
    seerr: "Seerr",
  };

  const typeColors: Record<string, string> = {
    radarr: "bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-400",
    sonarr: "bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-400",
    prowlarr: "bg-purple-100 text-purple-800 dark:bg-purple-900/40 dark:text-purple-400",
    seerr: "bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-400",
  };

  onMount(async () => {
    await loadServices();
  });

  async function loadServices() {
    try {
      loading = true;
      error = null;
      arrServices = await listArrServices();
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to load services";
    } finally {
      loading = false;
    }
  }

  function openAddModal() {
    editingService = null;
    showModal = true;
  }

  function openEditModal(service: ArrService) {
    editingService = service;
    showModal = true;
  }

  function closeModal() {
    showModal = false;
    editingService = null;
  }

  async function handleSave(data: ArrServiceInput) {
    if (editingService) {
      const updated = await updateArrService(editingService.id, data);
      arrServices = arrServices.map((s) => (s.id === updated.id ? updated : s));
    } else {
      const created = await createArrService(data);
      arrServices = [created, ...arrServices];
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Are you sure you want to remove this service?")) return;
    try {
      await deleteArrService(id);
      arrServices = arrServices.filter((s) => s.id !== id);
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to delete service";
    }
  }
</script>

<div>
  <div class="mb-6">
    <div class="flex items-center justify-between mb-2">
      <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">Services</h1>
    </div>
    <p class="text-sm text-slate-500 dark:text-slate-400">
      Manage your *arr integrations
    </p>
  </div>

  <div class="space-y-6">
    <!-- *arr Services Section -->
    <div class="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h2 class="text-lg font-semibold text-slate-900 dark:text-slate-100">*arr Services</h2>
          <p class="text-sm text-slate-500 dark:text-slate-400">
            Connect Radarr, Sonarr, Seerr and Prowlarr to manage your media
            library
          </p>
        </div>
        {#if isAdmin}
          <button
            onclick={openAddModal}
            class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors"
          >
            <Plus class="w-4 h-4" />
            Add Service
          </button>
        {/if}
      </div>

      {#if loading}
        <div class="flex items-center justify-center p-8">
          <div
            class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"
          ></div>
          <span class="ml-3 text-slate-600 dark:text-slate-400">Loading services...</span>
        </div>
      {:else if error}
        <div
          class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded-lg text-sm"
        >
          {error}
        </div>
      {:else if arrServices.length === 0}
        <div
          class="border-2 border-dashed border-slate-200 dark:border-slate-700 rounded-lg p-8 text-center"
        >
          <Server class="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
          <p class="text-sm text-slate-500 dark:text-slate-400 mb-4">No *arr services connected</p>
          {#if isAdmin}
            <button
              onclick={openAddModal}
              class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors"
            >
              <Plus class="w-4 h-4" />
              Add *arr Service
            </button>
          {:else}
            <p class="text-sm text-slate-400 dark:text-slate-500">Ask an admin to set up your *arr services.</p>
          {/if}
        </div>
      {:else}
        <div class="space-y-3">
          {#each arrServices as service (service.id)}
            <div
              class="flex items-center justify-between p-4 border border-slate-200 dark:border-slate-700 rounded-lg hover:border-slate-300 dark:hover:border-slate-600 transition-colors group"
            >
              <div class="flex items-center gap-4">
                <div
                  class="w-10 h-10 bg-slate-100 dark:bg-slate-700 rounded-lg flex items-center justify-center"
                >
                  <Server class="w-5 h-5 text-slate-600 dark:text-slate-400" />
                </div>
                <div>
                  <div class="flex items-center gap-2">
                    <h3 class="font-medium text-slate-900 dark:text-slate-100">
                      {service.name}
                    </h3>
                    <span
                      class="text-xs px-2 py-0.5 rounded-full font-medium {typeColors[
                        service.type
                      ]}"
                    >
                      {typeLabels[service.type]}
                    </span>
                  </div>
                  <p class="text-sm text-slate-500 dark:text-slate-400">{service.url}</p>
                </div>
              </div>
              <div
                class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <a
                  href={service.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  class="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
                  title="Open in browser"
                >
                  <ExternalLink class="w-4 h-4 text-slate-600 dark:text-slate-400" />
                </a>
                {#if isAdmin}
                  <button
                    onclick={() => openEditModal(service)}
                    class="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
                    title="Edit service"
                  >
                    <Edit2 class="w-4 h-4 text-slate-600 dark:text-slate-400" />
                  </button>
                  <button
                    onclick={() => handleDelete(service.id)}
                    class="p-2 hover:bg-red-50 dark:hover:bg-red-900/30 rounded-lg transition-colors"
                    title="Remove service"
                  >
                    <Trash2 class="w-4 h-4 text-red-600" />
                  </button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>

{#if showModal}
  <ArrServiceModal
    service={editingService}
    onSave={handleSave}
    onClose={closeModal}
  />
{/if}
