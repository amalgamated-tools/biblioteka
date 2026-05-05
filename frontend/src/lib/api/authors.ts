import type { Author, AuthorInput, PaginatedBooks } from "../../types";
import { request } from "./core";
import { listEntityBooks } from "./pagination";

interface AuthorListResponse {
  authors: Author[];
  total: number;
  limit: number;
  offset: number;
}

export async function listAuthors(): Promise<Author[]> {
  const result = await request<AuthorListResponse>("GET", "/api/authors");
  return result.authors;
}

export async function getAuthor(id: string): Promise<Author> {
  return request<Author>("GET", `/api/authors/${id}`);
}

export async function createAuthor(input: AuthorInput): Promise<Author> {
  return request<Author>("POST", "/api/authors", input);
}

export async function updateAuthor(
  id: string,
  input: AuthorInput,
): Promise<Author> {
  return request<Author>("PUT", `/api/authors/${id}`, input);
}

export async function deleteAuthor(id: string): Promise<void> {
  await request<void>("DELETE", `/api/authors/${id}`);
}

export async function listAuthorBooks(
  authorId: string,
  limit = 50,
  offset = 0,
): Promise<PaginatedBooks> {
  return listEntityBooks(`/api/authors/${authorId}`, limit, offset);
}
