import type { PaginatedBooks } from "../../types";
import { request } from "./core";

/**
 * Fetches a paginated list of books for a given entity path.
 *
 * Pass the entity base path (e.g. `/api/authors/${id}`) and the pagination
 * parameters; the `/books` suffix and query string are appended here.
 */
export async function listEntityBooks(
  entityPath: string,
  limit: number,
  offset: number,
): Promise<PaginatedBooks> {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  return request<PaginatedBooks>(
    "GET",
    `${entityPath}/books?${params.toString()}`,
  );
}
