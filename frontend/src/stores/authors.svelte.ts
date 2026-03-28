import type { Author, AuthorInput } from "../types";
import * as api from "../lib/api";

class AuthorStore {
  authors: Author[] = $state.raw([]);
  loading = $state(false);
  loaded = $state(false);

  async load(): Promise<void> {
    if (this.loading || this.loaded) return;
    this.loading = true;
    try {
      const data = await api.listAuthors();
      this.authors = data;
      this.loaded = true;
    } catch {
      // Silently fail — individual pages can handle errors
    } finally {
      this.loading = false;
    }
  }

  async add(input: AuthorInput): Promise<Author> {
    const created = await api.createAuthor(input);
    this.authors = [...this.authors, created];
    return created;
  }

  async edit(id: string, input: AuthorInput): Promise<Author> {
    const updated = await api.updateAuthor(id, input);
    this.authors = this.authors.map((a) => (a.id === id ? updated : a));
    return updated;
  }

  async remove(id: string): Promise<void> {
    await api.deleteAuthor(id);
    this.authors = this.authors.filter((a) => a.id !== id);
  }
}

export const authorStore = new AuthorStore();
