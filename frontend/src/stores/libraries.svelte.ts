import type { Library, LibraryInput } from "../types";
import { SvelteSet } from "svelte/reactivity";
import * as api from "../lib/api";

// Auto-clear scanning state after this duration as a safety net.
const SCANNING_TIMEOUT_MS = 5 * 60 * 1000;

class LibraryStore {
  libraries: Library[] = $state.raw([]);
  loading = $state(false);
  loaded = $state(false);
  // IDs of libraries whose background scan is in progress.
  scanningIds = new SvelteSet<string>();
  private scanningTimers = new Map<string, ReturnType<typeof setTimeout>>();
  isScanning = $derived(this.scanningIds.size > 0);

  clearScanning(id: string): void {
    const timer = this.scanningTimers.get(id);
    if (timer !== undefined) {
      clearTimeout(timer);
      this.scanningTimers.delete(id);
    }
    this.scanningIds.delete(id);
  }

  clearAllScanning(): void {
    for (const timer of this.scanningTimers.values()) {
      clearTimeout(timer);
    }
    this.scanningTimers.clear();
    this.scanningIds.clear();
  }

  async load(): Promise<void> {
    if (this.loading || this.loaded) return;
    this.loading = true;
    try {
      const data = await api.listLibraries();
      this.libraries = data;
      this.loaded = true;
    } catch {
      // Silently fail — individual pages can handle errors
    } finally {
      this.loading = false;
    }
  }

  async add(input: LibraryInput): Promise<Library> {
    const created = await api.createLibrary(input);
    this.libraries = [...this.libraries, created];
    // Mark the library as scanning so the UI can show a progress indicator
    // and poll for books until the background scan completes.
    this.scanningIds.add(created.id);
    const timer = setTimeout(
      () => this.clearScanning(created.id),
      SCANNING_TIMEOUT_MS,
    );
    this.scanningTimers.set(created.id, timer);
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
    this.clearScanning(id);
  }
}

export const libraryStore = new LibraryStore();
