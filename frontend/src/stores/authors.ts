import { writable } from "svelte/store";
import type { Author, AuthorInput } from "../types";
import * as api from "../lib/api";

export const authors = writable<Author[]>([]);
export const authorsLoading = writable(false);
export const authorsLoaded = writable(false);

export async function loadAuthors(): Promise<void> {
  authorsLoading.set(true);
  try {
    const data = await api.listAuthors();
    authors.set(data);
    authorsLoaded.set(true);
  } catch {
    // Silently fail — individual pages can handle errors
  } finally {
    authorsLoading.set(false);
  }
}

export async function addAuthor(input: AuthorInput): Promise<Author> {
  const created = await api.createAuthor(input);
  authors.update((list) => [...list, created]);
  return created;
}

export async function editAuthor(id: string, input: AuthorInput): Promise<Author> {
  const updated = await api.updateAuthor(id, input);
  authors.update((list) => list.map((a) => (a.id === id ? updated : a)));
  return updated;
}

export async function removeAuthor(id: string): Promise<void> {
  await api.deleteAuthor(id);
  authors.update((list) => list.filter((a) => a.id !== id));
}
