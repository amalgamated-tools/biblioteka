import type { ReadingList, ReadingListInput } from "../types";
import * as api from "../lib/api";

class ReadingListStore {
  lists: ReadingList[] = $state.raw([]);
  loading = $state(false);
  loaded = $state(false);

  async load(): Promise<void> {
    if (this.loading || this.loaded) return;
    this.loading = true;
    try {
      this.lists = await api.listReadingLists();
      this.loaded = true;
    } catch {
      // Silently fail — page components handle errors
    } finally {
      this.loading = false;
    }
  }

  /** Force a reload from the server, even if already loaded. */
  async reload(): Promise<void> {
    this.loaded = false;
    this.loading = false;
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
    // Increment book_count optimistically.
    this.lists = this.lists.map((l) =>
      l.id === listId ? { ...l, book_count: l.book_count + 1 } : l,
    );
  }

  async removeBook(listId: string, bookId: string): Promise<void> {
    await api.removeBookFromReadingList(listId, bookId);
    // Decrement book_count optimistically.
    this.lists = this.lists.map((l) =>
      l.id === listId
        ? { ...l, book_count: Math.max(0, l.book_count - 1) }
        : l,
    );
  }
}

export const readingListStore = new ReadingListStore();
