import type { ReadingList, ReadingListInput } from "../types";
import * as api from "../lib/api";

class ReadingListStore {
  lists: ReadingList[] = $state.raw([]);
  loading = $state(false);
  loaded = $state(false);
  loadError: string | null = $state(null);

  async load(): Promise<void> {
    if (this.loading || this.loaded) return;
    this.loading = true;
    this.loadError = null;
    try {
      this.lists = await api.listReadingLists();
      this.loaded = true;
    } catch (err) {
      this.loadError =
        err instanceof Error ? err.message : "failed to load reading lists";
      this.loaded = true;
    } finally {
      this.loading = false;
    }
  }

  /** Force a reload from the server, even if already loaded. */
  async reload(): Promise<void> {
    this.loaded = false;
    this.loading = false;
    this.loadError = null;
    return this.load();
  }

  async create(input: ReadingListInput): Promise<ReadingList> {
    const created = await api.createReadingList(input);
    this.lists = [...this.lists, created].sort((a, b) =>
      a.name.localeCompare(b.name),
    );
    return created;
  }

  async update(id: string, input: ReadingListInput): Promise<ReadingList> {
    const updated = await api.updateReadingList(id, input);
    this.lists = this.lists
      .map((l) => (l.id === id ? updated : l))
      .sort((a, b) => a.name.localeCompare(b.name));
    return updated;
  }

  async remove(id: string): Promise<void> {
    await api.deleteReadingList(id);
    this.lists = this.lists.filter((l) => l.id !== id);
  }

  async addBook(listId: string, bookId: string): Promise<void> {
    await api.addBookToReadingList(listId, bookId);
    // The backend add is idempotent, so a successful response does not
    // guarantee a new row was inserted. Re-sync from the server to keep
    // book_count accurate.
    await this.reload();
  }

  async removeBook(listId: string, bookId: string): Promise<void> {
    await api.removeBookFromReadingList(listId, bookId);
    // The backend remove is idempotent and may be a no-op. Re-sync from
    // the server to keep book_count accurate.
    await this.reload();
  }
}

export const readingListStore = new ReadingListStore();
