<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    variant: "error" | "success";
    children: Snippet;
    id?: string;
    role?: string;
    testId?: string;
    class?: string;
  }

  let {
    variant,
    children,
    id,
    role,
    testId,
    class: extraClass,
  }: Props = $props();

  const resolvedRole = $derived(
    role ?? (variant === "error" ? "alert" : "status"),
  );
  const liveRegion = $derived(
    resolvedRole === "alert" ? "assertive" : "polite",
  );

  const styles = {
    error:
      "bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400",
    success:
      "bg-success-50 dark:bg-green-900/20 border border-success-600/20 dark:border-green-700/30 text-success-700 dark:text-green-400",
  };
</script>

<div
  {id}
  class="px-4 py-3 rounded-xl text-sm animate-scale-in flex items-start gap-2 {styles[
    variant
  ]} {extraClass ?? ''}"
  role={resolvedRole}
  aria-live={liveRegion}
  data-testid={testId}
>
  {#if variant === "error"}
    <span
      class="inline-flex items-center justify-center w-4 h-4 mt-0.5 shrink-0 font-bold"
      aria-hidden="true"
      data-testid="alert-banner-error-icon">✕</span
    >
  {:else}
    <span
      class="inline-flex items-center justify-center w-4 h-4 mt-0.5 shrink-0 font-bold"
      aria-hidden="true"
      data-testid="alert-banner-success-icon">✓</span
    >
  {/if}
  <div class="flex-1 min-w-0">{@render children()}</div>
</div>
