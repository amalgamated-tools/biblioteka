<script lang="ts">
  import { getLLMConfig, setLLMConfig } from "../../lib/api";
  import type { LLMProvider } from "../../types";
  import { required, validate } from "../../lib/validation";
  import { BrainCircuit } from "lucide-svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";

  let form = $state<{
    provider: LLMProvider;
    endpoint: string;
    model: string;
    enabled: boolean;
  }>({
    provider: "ollama",
    endpoint: "",
    model: "",
    enabled: false,
  });

  let savedConfig = $state<{
    provider: LLMProvider;
    endpoint: string;
    model: string;
    enabled: boolean;
  }>({
    provider: "ollama",
    endpoint: "",
    model: "",
    enabled: false,
  });

  let loading = $state(false);
  let error: string | null = $state(null);
  let successMessage: string | null = $state(null);
  let restartRequired = $state(false);

  let configured = $derived(
    savedConfig.enabled &&
      savedConfig.endpoint.trim() !== "" &&
      savedConfig.model.trim() !== "",
  );

  let submitLabel = $derived(loading ? "Saving..." : "Save Configuration");

  $effect(() => {
    void (async () => {
      try {
        const config = await getLLMConfig();
        form.provider = config.provider || "ollama";
        form.endpoint = config.endpoint;
        form.model = config.model;
        form.enabled = config.enabled;
        savedConfig = {
          provider: config.provider || "ollama",
          endpoint: config.endpoint,
          model: config.model,
          enabled: config.enabled,
        };
      } catch {
        // ignore - user can re-enter
      }
    })();
  });

  async function handleSave(e: SubmitEvent) {
    e.preventDefault();
    error = null;
    successMessage = null;
    restartRequired = false;

    if (form.enabled) {
      error =
        validate(form.endpoint.trim(), [
          required("Endpoint is required when LLM is enabled"),
        ]) ??
        validate(form.model.trim(), [
          required("Model is required when LLM is enabled"),
        ]);
      if (error) return;
    }

    loading = true;

    try {
      const result = await setLLMConfig({
        provider: form.provider || "ollama",
        endpoint: form.endpoint.trim(),
        model: form.model.trim(),
        enabled: form.enabled,
      });
      savedConfig = {
        provider: form.provider || "ollama",
        endpoint: form.endpoint.trim(),
        model: form.model.trim(),
        enabled: form.enabled,
      };
      restartRequired = result.restart_required ?? false;
      successMessage = "LLM configuration saved successfully.";
    } catch (err) {
      error =
        err instanceof Error ? err.message : "Failed to save LLM configuration";
    } finally {
      loading = false;
    }
  }
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h2
        class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 flex items-center gap-2"
      >
        <BrainCircuit class="w-5 h-5 text-accent-600" aria-hidden="true" />
        LLM Configuration
      </h2>
    </div>

    <div class="mb-4">
      <div class="flex items-center gap-2 text-sm">
        <span class="text-ink-500 dark:text-ink-300">Status:</span>
        {#if configured}
          <span
            class="inline-flex items-center gap-1.5 text-success-700 dark:text-green-400 bg-success-50 dark:bg-green-900/20 px-2.5 py-1 rounded-full font-medium"
          >
            <span class="w-2 h-2 rounded-full bg-success-600"></span>
            Enabled
          </span>
        {:else}
          <span
            class="inline-flex items-center gap-1.5 text-ink-600 dark:text-ink-300 bg-ink-50 dark:bg-ink-800 px-2.5 py-1 rounded-full font-medium"
          >
            <span class="w-2 h-2 rounded-full bg-ink-300"></span>
            Disabled
          </span>
        {/if}
      </div>
    </div>

    <p class="text-sm text-ink-500 dark:text-ink-300 mb-4">
      Configure an LLM provider to enable AI-powered book metadata enrichment
      (genres, themes, mood, reading level, and suggested tags). Currently
      supports Ollama. Changes require a server restart to take effect.
    </p>

    <form onsubmit={handleSave} class="space-y-4">
      <div class="flex items-center gap-3">
        <input
          id="llm-enabled"
          type="checkbox"
          bind:checked={form.enabled}
          disabled={loading}
          class="h-4 w-4 rounded border-ink-400 text-accent-600 focus:ring-accent-500 focus:ring-2"
        />
        <label
          for="llm-enabled"
          class="text-sm font-medium text-ink-600 dark:text-ink-300"
        >
          Enable LLM enrichment
        </label>
      </div>

      <div>
        <label
          for="llm-provider"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Provider
        </label>
        <select
          id="llm-provider"
          bind:value={form.provider}
          disabled={loading}
          class="w-full px-4 py-2.5 border border-ink-400 dark:border-ink-400 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent focus-visible:outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
        >
          <option value="ollama">Ollama</option>
        </select>
      </div>

      <div>
        <label
          for="llm-endpoint"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Endpoint
        </label>
        <TextInput
          id="llm-endpoint"
          type="url"
          bind:value={form.endpoint}
          class="w-full py-2.5"
          placeholder="http://ollama.internal:11434"
          disabled={loading}
          aria-describedby="llm-endpoint-hint"
        />
        <p
          id="llm-endpoint-hint"
          class="text-xs text-ink-500 dark:text-ink-300 mt-1"
        >
          The base URL of the LLM API server, reachable from the Biblioteka
          server (loopback addresses are not allowed)
        </p>
      </div>

      <div>
        <label
          for="llm-model"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Model
        </label>
        <TextInput
          id="llm-model"
          type="text"
          bind:value={form.model}
          class="w-full py-2.5"
          placeholder="llama3"
          disabled={loading}
          aria-describedby="llm-model-hint"
        />
        <p
          id="llm-model-hint"
          class="text-xs text-ink-500 dark:text-ink-300 mt-1"
        >
          The model name to use for enrichment (e.g. llama3, mistral)
        </p>
      </div>

      {#if error}
        <AlertBanner variant="error">{error}</AlertBanner>
      {/if}

      {#if successMessage && !restartRequired}
        <AlertBanner variant="success">{successMessage}</AlertBanner>
      {/if}

      {#if restartRequired}
        <AlertBanner variant="warning">
          A server restart is required for the new LLM configuration to take
          effect.
        </AlertBanner>
      {/if}

      <Button type="submit" disabled={loading} class="w-full px-4 py-2.5">
        {submitLabel}
      </Button>
    </form>
  </div>
</div>
