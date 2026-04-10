<script lang="ts">
  import Button from "./Button.svelte";
  import { autofocusFirstButton } from "../../lib/actions";

  interface Props {
    itemId: string;
    itemName: string;
    onConfirm: () => void;
    onCancel: () => void;
    class?: string;
  }

  let {
    itemId,
    itemName,
    onConfirm,
    onCancel,
    class: extraClass,
  }: Props = $props();
</script>

<div
  class="flex items-center justify-end gap-2 animate-scale-in {extraClass ??
    ''}"
  role="group"
  aria-labelledby={`delete-confirm-label-${itemId}`}
  use:autofocusFirstButton
>
  <span
    id={`delete-confirm-label-${itemId}`}
    class="text-xs text-danger-600 dark:text-red-400">Delete "{itemName}"?</span
  >
  <Button
    type="button"
    variant="danger"
    onclick={onConfirm}
    class="px-3 py-1 text-xs"
  >
    Delete
  </Button>
  <Button
    type="button"
    variant="secondary"
    onclick={onCancel}
    class="px-3 py-1 text-xs"
  >
    Cancel
  </Button>
</div>
