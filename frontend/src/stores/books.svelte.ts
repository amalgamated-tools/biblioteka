import type { BookSummary, Book, BookInput } from "../types";
import * as api from "../lib/api";

class BookStore {
  books: BookSummary[] = $state.raw([]);
  loading = $state(false);
  loaded = $state(false);

  async load(): Promise<void> {
    this.loading = true;
    try {
      const all: BookSummary[] = [];
      let offset = 0;
      const limit = 200;
      for (;;) {
        const data = await api.listBooks(limit, offset);
        all.push(...data.books);
        if (all.length >= data.total) break;
        offset += limit;
      }
      this.books = all;
      this.loaded = true;
    } catch {
      // Silently fail — individual pages can handle errors
    } finally {
      this.loading = false;
    }
  }

  async add(input: BookInput): Promise<Book> {
    const created = await api.createBook(input);
    this.books = [...this.books, created];
    return created;
  }

  async edit(id: string, input: BookInput): Promise<Book> {
    const updated = await api.updateBook(id, input);
    this.books = this.books.map((b) => (b.id === id ? updated : b));
    return updated;
  }

  async remove(id: string): Promise<void> {
    await api.deleteBook(id);
    this.books = this.books.filter((b) => b.id !== id);
  }
}

export const bookStore = new BookStore();
