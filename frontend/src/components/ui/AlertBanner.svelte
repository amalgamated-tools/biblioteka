<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    variant: "error" | "success";
    children: Snippet;
    role?: string;
    testId?: string;
    class?: string;
  }

  let { variant, children, role, testId, class: extraClass }: Props = $props();

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
  class="px-4 py-3 rounded-xl text-sm animate-scale-in {styles[
    variant
  ]} {extraClass ?? ''}"
  role={resolvedRole}
  aria-live={liveRegion}
  data-testid={testId}
>
  {@render children()}
</div>
