<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";

  interface Props extends Omit<HTMLInputAttributes, "class"> {
    value?: string;
    type?: "text" | "email" | "password" | "url";
    class?: string;
  }

  let {
    value = $bindable(""),
    type = "text",
    disabled,
    class: extraClass,
    ...restProps
  }: Props = $props();
</script>

<input
  {...restProps}
  {type}
  bind:value
  {disabled}
  class="px-4 border border-ink-200 dark:border-ink-700 rounded-xl transition-all {disabled
    ? 'bg-ink-50 dark:bg-ink-800 text-ink-500 dark:text-ink-400 cursor-not-allowed'
    : 'bg-white text-ink-900 focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 placeholder:text-ink-300 dark:placeholder:text-ink-500'} {extraClass ??
    ''}"
/>
