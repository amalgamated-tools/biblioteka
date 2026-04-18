<script lang="ts">
  import type { PasskeyCredential } from "../../types";
  import { KeyRound, Trash2 } from "lucide-svelte";

  interface Props {
    passkeys: PasskeyCredential[];
    passkeyDeleting: string | null;
    onDelete: (id: string) => void;
  }

  let { passkeys, passkeyDeleting, onDelete }: Props = $props();
</script>

{#if passkeys.length > 0}
  <ul class="space-y-2 mb-4" aria-label="Registered passkeys">
    {#each passkeys as passkey (passkey.id)}
      <li
        class="flex items-center justify-between gap-2 p-3 rounded-xl border border-ink-100 dark:border-ink-700 bg-cream-50 dark:bg-ink-800"
      >
        <div class="flex items-center gap-2 min-w-0">
          <KeyRound
            class="w-4 h-4 shrink-0 text-accent-600"
            aria-hidden="true"
          />
          <span
            class="text-sm font-medium text-ink-800 dark:text-cream-100 truncate"
            >{passkey.name}</span
          >
          <span class="text-xs text-ink-500 dark:text-ink-300 shrink-0">
            {new Date(passkey.created_at).toLocaleDateString()}
          </span>
        </div>
        <button
          type="button"
          aria-label="Delete passkey {passkey.name}"
          disabled={passkeyDeleting === passkey.id}
          onclick={() => onDelete(passkey.id)}
          class="p-1.5 rounded-lg text-ink-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-40"
        >
          <Trash2 class="w-4 h-4" aria-hidden="true" />
        </button>
      </li>
    {/each}
  </ul>
{:else}
  <p class="text-sm text-ink-500 dark:text-ink-300 mb-4">
    No passkeys registered yet.
  </p>
{/if}
