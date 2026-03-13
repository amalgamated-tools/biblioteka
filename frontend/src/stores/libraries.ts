import { writable } from "svelte/store";
import type { Library, LibraryInput } from "../types";
import * as api from "../lib/api";

export const libraries = writable<Library[]>([]);
export const librariesLoading = writable(false);
export const librariesLoaded = writable(false);

export async function loadLibraries(): Promise<void> {
  librariesLoading.set(true);
  try {
    const data = await api.listLibraries();
    libraries.set(data);
    librariesLoaded.set(true);
  } catch {
    // Silently fail — individual pages can handle errors
  } finally {
    librariesLoading.set(false);
  }
}

export async function addLibrary(input: LibraryInput): Promise<Library> {
  const created = await api.createLibrary(input);
  libraries.update((libs) => [...libs, created]);
  return created;
}

export async function editLibrary(id: string, input: LibraryInput): Promise<Library> {
  const updated = await api.updateLibrary(id, input);
  libraries.update((libs) => libs.map((l) => (l.id === id ? updated : l)));
  return updated;
}

export async function removeLibrary(id: string): Promise<void> {
  await api.deleteLibrary(id);
  libraries.update((libs) => libs.filter((l) => l.id !== id));
}
