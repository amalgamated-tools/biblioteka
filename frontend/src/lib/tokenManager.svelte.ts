import { tick } from "svelte";
import { CopyTimeoutState } from "./copyTimeout.svelte";

export interface TokenManagerOps<T> {
  loadFn: () => Promise<T[]>;
  deleteFn: (id: string) => Promise<void>;
  loadErrorMessage: string;
  deleteErrorMessage: string;
}

/**
 * Creates a reactive token/key manager encapsulating the shared logic for
 * loading a list, confirming deletions with focus-restoration, and managing
 * a clipboard-copy timeout.
 *
 * Used by APIKeysTab and KoboTab to eliminate duplicated state and functions.
 */
export function createTokenManager<T extends { id: string }>(
  ops: TokenManagerOps<T>,
) {
  let items: T[] = $state.raw([]);
  let loading = $state(false);
  let error: string | null = $state(null);
  let pendingDelete: { id: string; name: string } | null = $state(null);
  const copyState = new CopyTimeoutState();

  async function load() {
    loading = true;
    error = null;
    try {
      items = await ops.loadFn();
    } catch (err) {
      error = err instanceof Error ? err.message : ops.loadErrorMessage;
    } finally {
      loading = false;
    }
  }

  function handleDelete(id: string, name: string) {
    pendingDelete = { id, name };
  }

  async function cancelDelete() {
    const id = pendingDelete?.id;
    pendingDelete = null;
    await tick();
    if (id) {
      const trigger = document.querySelector<HTMLElement>(
        `[data-delete-trigger="${id}"]`,
      );
      trigger?.focus();
    }
  }

  async function confirmDelete() {
    if (!pendingDelete) return;
    const { id } = pendingDelete;
    pendingDelete = null;
    error = null;
    try {
      await ops.deleteFn(id);
      items = items.filter((item) => item.id !== id);
    } catch (err) {
      error = err instanceof Error ? err.message : ops.deleteErrorMessage;
    }
  }

  function setCopied(id: string) {
    copyState.set(id);
  }

  return {
    get items() {
      return items;
    },
    set items(v: T[]) {
      items = v;
    },
    get loading() {
      return loading;
    },
    get error() {
      return error;
    },
    set error(v: string | null) {
      error = v;
    },
    get pendingDelete() {
      return pendingDelete;
    },
    get copiedId() {
      return copyState.copiedId;
    },
    clearCopyTimeout: () => copyState.clear(),
    load,
    handleDelete,
    cancelDelete,
    confirmDelete,
    setCopied,
  };
}
