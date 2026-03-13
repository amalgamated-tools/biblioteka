<script lang="ts">
  import { untrack } from "svelte";
  import { X } from "lucide-svelte";
  import type { ArrService, ArrServiceType } from "../types";

  interface Props {
    service: ArrService | null;
    onSave: (data: {
      name: string;
      type: ArrServiceType;
      url: string;
      api_key: string;
    }) => Promise<void>;
    onClose: () => void;
  }

  let { service, onSave, onClose }: Props = $props();

  let formData = $state(
    untrack(() => ({
      name: service?.name ?? "",
      type: (service?.type ?? "radarr") as ArrServiceType,
      url: service?.url ?? "",
      api_key: service?.api_key ?? "",
    })),
  );

  let loading = $state(false);
  let error: string | null = $state(null);

  const serviceTypes: { value: ArrServiceType; label: string }[] = [
    { value: "radarr", label: "Radarr" },
    { value: "sonarr", label: "Sonarr" },
    { value: "prowlarr", label: "Prowlarr" },
    { value: "seerr", label: "Seerr" },
  ];

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    error = null;

    if (!formData.name.trim()) {
      error = "Name is required";
      return;
    }
    if (!formData.url.trim()) {
      error = "URL is required";
      return;
    }
    if (!formData.api_key.trim()) {
      error = "API key is required";
      return;
    }

    loading = true;
    try {
      await onSave({
        name: formData.name,
        type: formData.type,
        url: formData.url,
        api_key: formData.api_key,
      });
      onClose();
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to save service";
    } finally {
      loading = false;
    }
  }
</script>

<div
  class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50"
>
  <div
    class="bg-white dark:bg-slate-800 rounded-2xl shadow-2xl dark:shadow-slate-900/30 w-full max-w-md"
  >
    <div
      class="border-b border-slate-200 dark:border-slate-700 px-6 py-4 flex items-center justify-between"
    >
      <h2 class="text-xl font-bold text-slate-900 dark:text-slate-100">
        {service ? "Edit Service" : "Add *arr Service"}
      </h2>
      <button
        onclick={onClose}
        class="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
      >
        <X class="w-5 h-5 text-slate-600 dark:text-slate-400" />
      </button>
    </div>

    <form onsubmit={handleSubmit} class="p-6 space-y-4">
      <div>
        <label
          for="service-name"
          class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
        >
          Name *
        </label>
        <input
          id="service-name"
          type="text"
          bind:value={formData.name}
          class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
          placeholder="My Radarr Server"
          disabled={loading}
          autofocus
        />
      </div>

      <div>
        <label
          for="service-type"
          class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
        >
          Type *
        </label>
        <select
          id="service-type"
          bind:value={formData.type}
          class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none bg-white"
          disabled={loading}
        >
          {#each serviceTypes as st (st.value)}
            <option value={st.value}>{st.label}</option>
          {/each}
        </select>
      </div>

      <div>
        <label
          for="service-url"
          class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
        >
          URL *
        </label>
        <input
          id="service-url"
          type="url"
          bind:value={formData.url}
          class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
          placeholder="http://localhost:7878"
          disabled={loading}
        />
        <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
          The base URL of your *arr instance (e.g. http://localhost:7878)
        </p>
      </div>

      <div>
        <label
          for="service-api-key"
          class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
        >
          API Key *
        </label>
        <input
          id="service-api-key"
          type="password"
          bind:value={formData.api_key}
          class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
          placeholder="Your API key"
          disabled={loading}
        />
        <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
          Found in your *arr instance under Settings &gt; General &gt; API Key
        </p>
      </div>

      {#if error}
        <div
          class="bg-red-50 border border-red-200 text-red-700 dark:bg-red-900/30 dark:border-red-800 dark:text-red-400 px-4 py-3 rounded-lg text-sm"
        >
          {error}
        </div>
      {/if}

      <div class="flex gap-3 pt-2">
        <button
          type="button"
          onclick={onClose}
          class="flex-1 px-4 py-3 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-300 font-medium rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
          disabled={loading}
        >
          Cancel
        </button>
        <button
          type="submit"
          class="flex-1 px-4 py-3 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
          disabled={loading}
        >
          {loading ? "Saving..." : service ? "Update Service" : "Add Service"}
        </button>
      </div>
    </form>
  </div>
</div>
