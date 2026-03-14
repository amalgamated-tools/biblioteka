import type { Library, LibraryInput } from "../types";
import * as api from "../lib/api";

class LibraryStore {
  libraries: Library[] = $state.raw([]);
  async load(): Promise<void> {
    if (this.loading || this.loaded) return;
    this.loading = true;
    }
  }

  async add(input: LibraryInput): Promise<Library> {
    const created = await api.createLibrary(input);
    this.libraries = [...this.libraries, created];
    return created;
  }

  async edit(id: string, input: LibraryInput): Promise<Library> {
    const updated = await api.updateLibrary(id, input);
    this.libraries = this.libraries.map((l) => (l.id === id ? updated : l));
    return updated;
  }

  async remove(id: string): Promise<void> {
    await api.deleteLibrary(id);
    this.libraries = this.libraries.filter((l) => l.id !== id);
  }
}

export const libraryStore = new LibraryStore();
