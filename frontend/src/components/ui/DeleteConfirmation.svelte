<script lang="ts">
  import Button from "./Button.svelte";
  import { autofocusFirstButton } from "../../lib/actions";

  interface Props {
    itemId: string;
    itemName: string;
    onconfirm: () => void;
    oncancel: () => void;
    class?: string;
  }

  let {
    itemId,
    itemName,
    onconfirm,
    oncancel,
    class: extraClass,
  }: Props = $props();
</script>

<div
  class="flex items-center gap-2 animate-scale-in {extraClass ?? ''}"
  role="alertdialog"
  aria-modal="false"
  aria-labelledby={`delete-confirm-label-${itemId}`}
  tabindex="-1"
  use:autofocusFirstButton
  onkeydown={(e: KeyboardEvent) => {
    if (e.key === "Escape") oncancel();
  }}
>
  <span
    id={`delete-confirm-label-${itemId}`}
    class="text-xs text-danger-600 dark:text-red-400"
    >Delete "{itemName}"?</span
  >
  <Button
    type="button"
    variant="danger"
    onclick={onconfirm}
    class="px-3 py-1 text-xs"
  >
    Delete
  </Button>
  <Button
    type="button"
    variant="secondary"
    onclick={oncancel}
    class="px-3 py-1 text-xs"
  >
    Cancel
  </Button>
</div>
