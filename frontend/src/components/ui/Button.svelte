<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    variant?: "primary" | "secondary" | "danger";
    type?: "button" | "submit" | "reset";
    disabled?: boolean;
    class?: string;
    onclick?: (e: MouseEvent) => void;
    children: Snippet;
  }

  let {
    variant = "primary",
    type = "button",
    disabled = false,
    class: extraClass,
    onclick,
    children,
  }: Props = $props();

  const variantClasses = {
    primary:
      "bg-gradient-to-r from-accent-600 to-accent-700 hover:from-accent-700 hover:to-accent-800 text-white shadow-md shadow-accent-600/20",
    secondary:
      "border border-ink-200 dark:border-ink-700 text-ink-600 dark:text-ink-300 hover:bg-ink-50 dark:hover:bg-ink-800",
    danger: "bg-danger-600 text-white hover:bg-danger-700",
  };
</script>

<button
  {type}
  {disabled}
  {onclick}
  class="font-semibold rounded-xl transition-all disabled:opacity-50 disabled:cursor-not-allowed {variantClasses[
    variant
  ]} {extraClass ?? ''}"
>
  {@render children()}
</button>
