import { writable } from "svelte/store";
import type { BookSummary, Book, BookInput } from "../types";
import * as api from "../lib/api";

export const books = writable<BookSummary[]>([]);
export const booksLoading = writable(false);
export const booksLoaded = writable(false);

export async function loadBooks(): Promise<void> {
  booksLoading.set(true);
  try {
    const data = await api.listBooks();
    books.set(data);
    booksLoaded.set(true);
  } catch {
    // Silently fail — individual pages can handle errors
  } finally {
    booksLoading.set(false);
  }
}

export async function addBook(input: BookInput): Promise<Book> {
  const created = await api.createBook(input);
  books.update((list) => [...list, created]);
  return created;
}

export async function editBook(id: string, input: BookInput): Promise<Book> {
  const updated = await api.updateBook(id, input);
  books.update((list) => list.map((b) => (b.id === id ? updated : b)));
  return updated;
}

export async function removeBook(id: string): Promise<void> {
  await api.deleteBook(id);
  books.update((list) => list.filter((b) => b.id !== id));
}
