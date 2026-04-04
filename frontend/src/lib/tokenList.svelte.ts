import { tick } from "svelte";
import { CopyTimeoutState } from "./copyTimeout.svelte";

export interface TokenListOps<T extends { id: string }> {
  load: () => Promise<T[]>;
  delete: (id: string) => Promise<void>;
  loadError: string;
  deleteError: string;
}

/**
 * Manages the load/delete lifecycle for a list of token-like resources
 * (API keys, Kobo tokens, etc.) and exposes copy-to-clipboard feedback
 * via the `copy` field.
 *
 * State is reactive via Svelte 5 `$state` runes.
 */
export class TokenListState<T extends { id: string }> {
  items: T[] = $state.raw([]);
  loading = $state(false);
  error: string | null = $state(null);
  pendingDelete: { id: string; name: string } | null = $state(null);
  readonly copy = new CopyTimeoutState();

  private readonly ops: TokenListOps<T>;

  constructor(ops: TokenListOps<T>) {
    this.ops = ops;
  }

  async load(): Promise<void> {
    this.loading = true;
    this.error = null;
    try {
      this.items = await this.ops.load();
    } catch (err) {
      this.error = err instanceof Error ? err.message : this.ops.loadError;
    } finally {
      this.loading = false;
    }
  }

  handleDelete(id: string, name: string): void {
    this.pendingDelete = { id, name };
  }

  async cancelDelete(onAfterClear?: () => void): Promise<void> {
    this.pendingDelete = null;
    await tick();
    onAfterClear?.();
  }

  cancelDeleteWithFocus = (): void => {
    const id = this.pendingDelete?.id;
    void this.cancelDelete(
      id
        ? () =>
            document
              .querySelector<HTMLElement>(`[data-delete-trigger="${id}"]`)
              ?.focus()
        : undefined,
    );
  };

  confirmDelete = async (): Promise<void> => {
    if (!this.pendingDelete) return;
    const { id } = this.pendingDelete;
    this.pendingDelete = null;
    this.error = null;
    try {
      await this.ops.delete(id);
      this.items = this.items.filter((item) => item.id !== id);
    } catch (err) {
      this.error = err instanceof Error ? err.message : this.ops.deleteError;
    }
  }
}
