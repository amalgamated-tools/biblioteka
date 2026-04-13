import type {
  PaginatedBooks,
  ReadingList,
  ReadingListInput,
} from "../../types";
import { request } from "./core";
import { listEntityBooks } from "./pagination";

export async function listReadingLists(): Promise<ReadingList[]> {
  return request<ReadingList[]>("GET", "/api/reading-lists");
}

export async function getReadingList(id: string): Promise<ReadingList> {
  return request<ReadingList>("GET", `/api/reading-lists/${id}`);
}

export async function createReadingList(
  input: ReadingListInput,
): Promise<ReadingList> {
  return request<ReadingList>("POST", "/api/reading-lists", input);
}

export async function updateReadingList(
  id: string,
  input: ReadingListInput,
): Promise<ReadingList> {
  return request<ReadingList>("PUT", `/api/reading-lists/${id}`, input);
}

export async function deleteReadingList(id: string): Promise<void> {
  await request<void>("DELETE", `/api/reading-lists/${id}`);
}

export async function listReadingListBooks(
  listId: string,
  limit = 50,
  offset = 0,
): Promise<PaginatedBooks> {
  return listEntityBooks(`/api/reading-lists/${listId}`, limit, offset);
}

export async function addBookToReadingList(
  listId: string,
  bookId: string,
): Promise<void> {
  await request<void>("POST", `/api/reading-lists/${listId}/books`, {
    book_id: bookId,
  });
}

export async function removeBookFromReadingList(
  listId: string,
  bookId: string,
): Promise<void> {
  await request<void>("DELETE", `/api/reading-lists/${listId}/books/${bookId}`);
}

export async function getReadingListsForBook(
  bookId: string,
): Promise<ReadingList[]> {
  return request<ReadingList[]>("GET", `/api/books/${bookId}/reading-lists`);
}
