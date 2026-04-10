import type { Library, LibraryInput, PaginatedBooks } from "../../types";
import { request } from "./core";
import { listEntityBooks } from "./pagination";

export async function listLibraries(): Promise<Library[]> {
  return request<Library[]>("GET", "/api/libraries");
}

export async function createLibrary(input: LibraryInput): Promise<Library> {
  return request<Library>("POST", "/api/libraries", input);
}

export async function updateLibrary(
  id: string,
  input: LibraryInput,
): Promise<Library> {
  return request<Library>("PUT", `/api/libraries/${id}`, input);
}

export async function deleteLibrary(id: string): Promise<void> {
  await request<void>("DELETE", `/api/libraries/${id}`);
}

export async function listLibraryBooks(
  libraryId: string,
  limit = 50,
  offset = 0,
): Promise<PaginatedBooks> {
  return listEntityBooks(`/api/libraries/${libraryId}`, limit, offset);
}
