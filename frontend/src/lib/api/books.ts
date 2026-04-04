import type {
  Author,
  Book,
  BookInput,
  BookSeriesEntry,
  BookFile,
  BookFileInput,
  PaginatedBooks,
} from "../../types";
import { request } from "./core";

export async function listBooks(
  limit = 50,
  offset = 0,
  query = "",
): Promise<PaginatedBooks> {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  if (query) {
    params.set("query", query);
  }
  return request<PaginatedBooks>("GET", `/api/books?${params.toString()}`);
}

export async function getBook(id: string): Promise<Book> {
  return request<Book>("GET", `/api/books/${id}`);
}

export async function createBook(input: BookInput): Promise<Book> {
  return request<Book>("POST", "/api/books", input);
}

export async function updateBook(id: string, input: BookInput): Promise<Book> {
  return request<Book>("PUT", `/api/books/${id}`, input);
}

export async function deleteBook(id: string): Promise<void> {
  await request<void>("DELETE", `/api/books/${id}`);
}

export async function getBookAuthors(bookId: string): Promise<Author[]> {
  return request<Author[]>("GET", `/api/books/${bookId}/authors`);
}

export async function setBookAuthors(
  bookId: string,
  authorIds: string[],
): Promise<Author[]> {
  return request<Author[]>("PUT", `/api/books/${bookId}/authors`, {
    author_ids: authorIds,
  });
}

export async function getBookSeries(
  bookId: string,
): Promise<BookSeriesEntry[]> {
  return request<BookSeriesEntry[]>("GET", `/api/books/${bookId}/series`);
}

export async function setBookSeries(
  bookId: string,
  entries: { series_id: string; position?: number }[],
): Promise<BookSeriesEntry[]> {
  return request<BookSeriesEntry[]>("PUT", `/api/books/${bookId}/series`, {
    entries,
  });
}

export async function listBookFiles(bookId: string): Promise<BookFile[]> {
  return request<BookFile[]>("GET", `/api/books/${bookId}/files`);
}

export async function createBookFile(
  bookId: string,
  input: BookFileInput,
): Promise<BookFile> {
  return request<BookFile>("POST", `/api/books/${bookId}/files`, input);
}

export async function getBookFile(id: string): Promise<BookFile> {
  return request<BookFile>("GET", `/api/book-files/${id}`);
}

export async function deleteBookFile(id: string): Promise<void> {
  await request<void>("DELETE", `/api/book-files/${id}`);
}
